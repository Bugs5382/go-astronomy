package satellite

/*
MIT License

Copyright (c) 2026 Shane

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

// SGP4 near-Earth propagation and the initialisation shared with SDP4.
//
// Ported from the public-domain reference implementation of Vallado,
// Crawford, Hujsak and Kelso (2006), by way of its Python transliteration
// in brandon-rhodes/python-sgp4 (MIT).
//
// Reference: D. A. Vallado, P. Crawford, R. Hujsak and T. S. Kelso,
// "Revisiting Spacetrack Report #3", AIAA 2006-6753.
//
// The port keeps the structure and the variable names of the reference
// code (sgp4init, initl, dscom, dpper, dsinit, dspace, sgp4 and gstime) so
// the two can be read side by side. Only the WGS-72 constants and the
// "improved" operation mode are carried over: TLEs are fitted against
// WGS-72, and the improved mode is what the published verification output
// (tcppver.out) was produced with.
//
// One deliberate difference: the reference keeps the resonance integrator
// state (atime, xli, xni) in the satellite record between calls as a speed
// optimisation. Here the propagator state is read-only after
// initialisation, so the integrator always restarts from epoch. The result
// is the same, and an Elements value can be shared between goroutines.
//
// The deep-space (SDP4) routines, dscom, dpper, dsinit, and dspace, are in
// sdp4.go.

import "math"

// satrec holds the initialised propagator state, mirroring the elsetrec
// structure of the reference code.
type satrec struct {
	// Mean elements at epoch, in radians and radians per minute.
	bstar, ecco, argpo, inclo, mo, noKozai, nodeo float64
	noUnkozai                                     float64

	// Near-earth coefficients.
	isimp                                   int
	method                                  byte // 'n' near earth, 'd' deep space
	aycof, con41, cc1, cc4, cc5, d2, d3, d4 float64
	delmo, eta, argpdot, omgcof, sinmao     float64
	t2cof, t3cof, t4cof, t5cof              float64
	x1mth2, x7thm1, mdot, nodedot, xlcof    float64
	xmcof, nodecf                           float64

	// Deep-space coefficients.
	irez                                           int
	d2201, d2211, d3210, d3222, d4410, d4422       float64
	d5220, d5232, d5421, d5433                     float64
	dedt, del1, del2, del3, didt, dmdt, dnodt      float64
	domdt, e3, ee2, peo, pgho, pho, pinco, plo     float64
	se2, se3, sgh2, sgh3, sgh4, sh2, sh3, si2, si3 float64
	sl2, sl3, sl4, gsto, xfact                     float64
	xgh2, xgh3, xgh4, xh2, xh3, xi2, xi3           float64
	xl2, xl3, xl4, xlamo, zmol, zmos, xli, xni     float64
}

// initl un-Kozais the mean motion and computes the epoch quantities shared
// by the near-earth and deep-space set-up.
type initlOut struct {
	ainv, ao, con42, cosio, cosio2, eccsq, omeosq, posq, rp, rteosq, sinio float64
}

func initl(r *satrec, epoch float64) initlOut {
	var o initlOut
	ecco := r.ecco

	o.eccsq = ecco * ecco
	o.omeosq = 1.0 - o.eccsq
	o.rteosq = math.Sqrt(o.omeosq)
	o.cosio = math.Cos(r.inclo)
	o.cosio2 = o.cosio * o.cosio

	// Un-Kozai the mean motion.
	ak := math.Pow(wgs72Xke/r.noKozai, x2o3)
	d1 := 0.75 * wgs72J2 * (3.0*o.cosio2 - 1.0) / (o.rteosq * o.omeosq)
	del := d1 / (ak * ak)
	adel := ak * (1.0 - del*del - del*
		(1.0/3.0+134.0*del*del/81.0))
	del = d1 / (adel * adel)
	r.noUnkozai = r.noKozai / (1.0 + del)

	o.ao = math.Pow(wgs72Xke/r.noUnkozai, x2o3)
	o.sinio = math.Sin(r.inclo)
	po := o.ao * o.omeosq
	o.con42 = 1.0 - 5.0*o.cosio2
	r.con41 = -o.con42 - o.cosio2 - o.cosio2
	o.ainv = 1.0 / o.ao
	o.posq = po * po
	o.rp = o.ao * (1.0 - ecco)
	r.method = 'n'

	r.gsto = gmst82(epoch + jd1950)
	return o
}

// sgp4init fills in the propagator state from the mean elements already set
// on r. epoch is in days since 0 January 1950, 0h UTC.
func sgp4init(r *satrec, epoch float64) {
	const temp4 = 1.5e-12

	ss := 78.0/wgs72RadiusEarthKm + 1.0
	qzms2ttemp := (120.0 - 78.0) / wgs72RadiusEarthKm
	qzms2t := qzms2ttemp * qzms2ttemp * qzms2ttemp * qzms2ttemp

	r.method = 'n'
	in := initl(r, epoch)
	ao, cosio, cosio2, sinio := in.ao, in.cosio, in.cosio2, in.sinio
	omeosq, rteosq, con42 := in.omeosq, in.rteosq, in.con42

	// The reference dropped its sub-orbital (rp < 1) check: the radius test
	// in sgp4 catches decaying objects even when they start below the
	// surface.
	if omeosq >= 0.0 || r.noUnkozai >= 0.0 {
		r.isimp = 0
		if in.rp < 220.0/wgs72RadiusEarthKm+1.0 {
			r.isimp = 1
		}
		sfour := ss
		qzms24 := qzms2t
		perige := (in.rp - 1.0) * wgs72RadiusEarthKm

		// For perigees below 156 km, s and qoms2t are altered.
		if perige < 156.0 {
			sfour = perige - 78.0
			if perige < 98.0 {
				sfour = 20.0
			}
			qzms24temp := (120.0 - sfour) / wgs72RadiusEarthKm
			qzms24 = qzms24temp * qzms24temp * qzms24temp * qzms24temp
			sfour = sfour/wgs72RadiusEarthKm + 1.0
		}
		pinvsq := 1.0 / in.posq

		tsi := 1.0 / (ao - sfour)
		r.eta = ao * r.ecco * tsi
		etasq := r.eta * r.eta
		eeta := r.ecco * r.eta
		psisq := math.Abs(1.0 - etasq)
		coef := qzms24 * math.Pow(tsi, 4.0)
		coef1 := coef / math.Pow(psisq, 3.5)
		cc2 := coef1 * r.noUnkozai * (ao*(1.0+1.5*etasq+eeta*
			(4.0+etasq)) + 0.375*wgs72J2*tsi/psisq*r.con41*
			(8.0+3.0*etasq*(8.0+etasq)))
		r.cc1 = r.bstar * cc2
		cc3 := 0.0
		if r.ecco > 1.0e-4 {
			cc3 = -2.0 * coef * tsi * wgs72J3oJ2 * r.noUnkozai * sinio / r.ecco
		}
		r.x1mth2 = 1.0 - cosio2
		r.cc4 = 2.0 * r.noUnkozai * coef1 * ao * omeosq *
			(r.eta*(2.0+0.5*etasq) + r.ecco*
				(0.5+2.0*etasq) - wgs72J2*tsi/(ao*psisq)*
				(-3.0*r.con41*(1.0-2.0*eeta+etasq*
					(1.5-0.5*eeta))+0.75*r.x1mth2*
					(2.0*etasq-eeta*(1.0+etasq))*math.Cos(2.0*r.argpo)))
		r.cc5 = 2.0 * coef1 * ao * omeosq * (1.0 + 2.75*
			(etasq+eeta) + eeta*etasq)
		cosio4 := cosio2 * cosio2
		temp1 := 1.5 * wgs72J2 * pinvsq * r.noUnkozai
		temp2 := 0.5 * temp1 * wgs72J2 * pinvsq
		temp3 := -0.46875 * wgs72J4 * pinvsq * pinvsq * r.noUnkozai
		r.mdot = r.noUnkozai + 0.5*temp1*rteosq*r.con41 + 0.0625*
			temp2*rteosq*(13.0-78.0*cosio2+137.0*cosio4)
		r.argpdot = (-0.5*temp1*con42 + 0.0625*temp2*
			(7.0-114.0*cosio2+395.0*cosio4) +
			temp3*(3.0-36.0*cosio2+49.0*cosio4))
		xhdot1 := -temp1 * cosio
		r.nodedot = xhdot1 + (0.5*temp2*(4.0-19.0*cosio2)+
			2.0*temp3*(3.0-7.0*cosio2))*cosio
		xpidot := r.argpdot + r.nodedot
		r.omgcof = r.bstar * cc3 * math.Cos(r.argpo)
		r.xmcof = 0.0
		if r.ecco > 1.0e-4 {
			r.xmcof = -x2o3 * coef * r.bstar / eeta
		}
		r.nodecf = 3.5 * omeosq * xhdot1 * r.cc1
		r.t2cof = 1.5 * r.cc1
		// Guard against dividing by zero at 180 degrees inclination.
		if math.Abs(cosio+1.0) > 1.5e-12 {
			r.xlcof = -0.25 * wgs72J3oJ2 * sinio * (3.0 + 5.0*cosio) / (1.0 + cosio)
		} else {
			r.xlcof = -0.25 * wgs72J3oJ2 * sinio * (3.0 + 5.0*cosio) / temp4
		}
		r.aycof = -0.5 * wgs72J3oJ2 * sinio
		delmotemp := 1.0 + r.eta*math.Cos(r.mo)
		r.delmo = delmotemp * delmotemp * delmotemp
		r.sinmao = math.Sin(r.mo)
		r.x7thm1 = 7.0*cosio2 - 1.0

		// Deep-space initialisation for periods of 225 minutes or more.
		if 2*math.Pi/r.noUnkozai >= 225.0 {
			r.method = 'd'
			r.isimp = 1
			tc := 0.0
			inclm := r.inclo

			d := dscom(r, epoch, r.ecco, r.argpo, tc, r.inclo, r.nodeo, r.noUnkozai)
			r.ecco, r.inclo, r.nodeo, r.argpo, r.mo = dpper(r, 0.0, true,
				r.ecco, r.inclo, r.nodeo, r.argpo, r.mo)
			dsinit(r, d, 0.0, tc, xpidot, in.eccsq, inclm)
		}

		// Remaining near-earth drag terms.
		if r.isimp != 1 {
			cc1sq := r.cc1 * r.cc1
			r.d2 = 4.0 * ao * tsi * cc1sq
			temp := r.d2 * tsi * r.cc1 / 3.0
			r.d3 = (17.0*ao + sfour) * temp
			r.d4 = 0.5 * temp * ao * tsi * (221.0*ao + 31.0*sfour) *
				r.cc1
			r.t3cof = r.d2 + 2.0*cc1sq
			r.t4cof = 0.25 * (3.0*r.d3 + r.cc1*
				(12.0*r.d2+10.0*cc1sq))
			r.t5cof = 0.2 * (3.0*r.d4 +
				12.0*r.cc1*r.d3 +
				6.0*r.d2*r.d2 +
				15.0*cc1sq*(2.0*r.d2+cc1sq))
		}
	}
	// The reference finishes with a call to sgp4 at tsince zero only to set
	// its error flag. Nothing here depends on that call, so it is left to
	// Propagate to report problems.
}

// sgp4 is the prediction model: secular gravity and drag, the deep-space
// terms where they apply, Kepler's equation, then the short-period
// periodics. It returns TEME position (km) and velocity (km/s).
func sgp4(r *satrec, tsince float64) (pos, vel [3]float64, err error) {
	const temp4 = 1.5e-12
	vkmpersec := wgs72RadiusEarthKm * wgs72Xke / 60.0

	t := tsince

	// Secular gravity and atmospheric drag.
	xmdf := r.mo + r.mdot*t
	argpdf := r.argpo + r.argpdot*t
	nodedf := r.nodeo + r.nodedot*t
	argpm := argpdf
	mm := xmdf
	t2 := t * t
	nodem := nodedf + r.nodecf*t2
	tempa := 1.0 - r.cc1*t
	tempe := r.bstar * r.cc4 * t
	templ := r.t2cof * t2

	if r.isimp != 1 {
		delomg := r.omgcof * t
		delmtemp := 1.0 + r.eta*math.Cos(xmdf)
		delm := r.xmcof *
			(delmtemp*delmtemp*delmtemp -
				r.delmo)
		temp := delomg + delm
		mm = xmdf + temp
		argpm = argpdf - temp
		t3 := t2 * t
		t4 := t3 * t
		tempa = tempa - r.d2*t2 - r.d3*t3 -
			r.d4*t4
		tempe = tempe + r.bstar*r.cc5*(math.Sin(mm)-
			r.sinmao)
		templ = templ + r.t3cof*t3 + t4*(r.t4cof+
			t*r.t5cof)
	}

	nm := r.noUnkozai
	em := r.ecco
	inclm := r.inclo
	if r.method == 'd' {
		tc := t
		em, argpm, inclm, mm, nodem, nm = dspace(r, t, tc, em, argpm, inclm, mm, nodem, nm)
	}

	if nm <= 0.0 {
		return pos, vel, propErr(2, tsince, nm)
	}

	am := math.Pow(wgs72Xke/nm, x2o3) * tempa * tempa
	nm = wgs72Xke / math.Pow(am, 1.5)
	em = em - tempe

	if em >= 1.0 || em < -0.001 {
		return pos, vel, propErr(1, tsince, em)
	}

	// Avoid a divide by zero for circular orbits.
	if em < 1.0e-6 {
		em = 1.0e-6
	}
	mm = mm + r.noUnkozai*templ
	xlm := mm + argpm + nodem

	nodem = math.Mod(nodem, twoPi)
	argpm = math.Mod(argpm, twoPi)
	xlm = math.Mod(xlm, twoPi)
	mm = math.Mod(xlm-argpm-nodem, twoPi)

	sinim := math.Sin(inclm)
	cosim := math.Cos(inclm)

	// Lunar-solar periodics.
	ep := em
	xincp := inclm
	argpp := argpm
	nodep := nodem
	mp := mm
	sinip := sinim
	cosip := cosim
	aycof, xlcof := r.aycof, r.xlcof
	con41, x1mth2, x7thm1 := r.con41, r.x1mth2, r.x7thm1
	if r.method == 'd' {
		ep, xincp, nodep, argpp, mp = dpper(r, t, false, ep, xincp, nodep, argpp, mp)
		if xincp < 0.0 {
			xincp = -xincp
			nodep = nodep + math.Pi
			argpp = argpp - math.Pi
		}
		if ep < 0.0 || ep > 1.0 {
			return pos, vel, propErr(3, tsince, ep)
		}

		// Long-period periodics with the perturbed inclination.
		sinip = math.Sin(xincp)
		cosip = math.Cos(xincp)
		aycof = -0.5 * wgs72J3oJ2 * sinip
		if math.Abs(cosip+1.0) > 1.5e-12 {
			xlcof = -0.25 * wgs72J3oJ2 * sinip * (3.0 + 5.0*cosip) / (1.0 + cosip)
		} else {
			xlcof = -0.25 * wgs72J3oJ2 * sinip * (3.0 + 5.0*cosip) / temp4
		}
	}

	axnl := ep * math.Cos(argpp)
	temp := 1.0 / (am * (1.0 - ep*ep))
	aynl := ep*math.Sin(argpp) + temp*aycof
	xl := mp + argpp + nodep + temp*xlcof*axnl

	// Kepler's equation, with the step limited to 0.95 rad.
	u := math.Mod(xl-nodep, twoPi)
	eo1 := u
	tem5 := 9999.9
	var sineo1, coseo1 float64
	for ktr := 1; math.Abs(tem5) >= 1.0e-12 && ktr <= 10; ktr++ {
		sineo1 = math.Sin(eo1)
		coseo1 = math.Cos(eo1)
		tem5 = 1.0 - coseo1*axnl - sineo1*aynl
		tem5 = (u - aynl*coseo1 + axnl*sineo1 - eo1) / tem5
		if math.Abs(tem5) >= 0.95 {
			if tem5 > 0.0 {
				tem5 = 0.95
			} else {
				tem5 = -0.95
			}
		}
		eo1 = eo1 + tem5
	}

	// Short-period preliminary quantities.
	ecose := axnl*coseo1 + aynl*sineo1
	esine := axnl*sineo1 - aynl*coseo1
	el2 := axnl*axnl + aynl*aynl
	pl := am * (1.0 - el2)
	if pl < 0.0 {
		return pos, vel, propErr(4, tsince, pl)
	}

	rl := am * (1.0 - ecose)
	rdotl := math.Sqrt(am) * esine / rl
	rvdotl := math.Sqrt(pl) / rl
	betal := math.Sqrt(1.0 - el2)
	temp = esine / (1.0 + betal)
	sinu := am / rl * (sineo1 - aynl - axnl*temp)
	cosu := am / rl * (coseo1 - axnl + aynl*temp)
	su := math.Atan2(sinu, cosu)
	sin2u := (cosu + cosu) * sinu
	cos2u := 1.0 - 2.0*sinu*sinu
	temp = 1.0 / pl
	temp1 := 0.5 * wgs72J2 * temp
	temp2 := temp1 * temp

	// Short-period periodics.
	if r.method == 'd' {
		cosisq := cosip * cosip
		con41 = 3.0*cosisq - 1.0
		x1mth2 = 1.0 - cosisq
		x7thm1 = 7.0*cosisq - 1.0
	}

	mrt := rl*(1.0-1.5*temp2*betal*con41) +
		0.5*temp1*x1mth2*cos2u
	su = su - 0.25*temp2*x7thm1*sin2u
	xnode := nodep + 1.5*temp2*cosip*sin2u
	xinc := xincp + 1.5*temp2*cosip*sinip*cos2u
	mvt := rdotl - nm*temp1*x1mth2*sin2u/wgs72Xke
	rvdot := rvdotl + nm*temp1*(x1mth2*cos2u+
		1.5*con41)/wgs72Xke

	// Orientation vectors.
	sinsu := math.Sin(su)
	cossu := math.Cos(su)
	snod := math.Sin(xnode)
	cnod := math.Cos(xnode)
	sini := math.Sin(xinc)
	cosi := math.Cos(xinc)
	xmx := -snod * cosi
	xmy := cnod * cosi
	ux := xmx*sinsu + cnod*cossu
	uy := xmy*sinsu + snod*cossu
	uz := sini * sinsu
	vx := xmx*cossu - cnod*sinsu
	vy := xmy*cossu - snod*sinsu
	vz := sini * cossu

	// Position and velocity in km and km/s.
	mr := mrt * wgs72RadiusEarthKm
	pos = [3]float64{mr * ux, mr * uy, mr * uz}
	vel = [3]float64{
		(mvt*ux + rvdot*vx) * vkmpersec,
		(mvt*uy + rvdot*vy) * vkmpersec,
		(mvt*uz + rvdot*vz) * vkmpersec,
	}

	// A radius below one earth radius means the object has come down. The
	// vectors are still returned, as the reference does.
	if mrt < 1.0 {
		return pos, vel, propErr(6, tsince, mrt)
	}
	return pos, vel, nil
}
