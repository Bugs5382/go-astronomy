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

// SDP4 deep-space propagation: the lunar-solar periodics (dpper), the
// deep-space common terms (dscom), and the resonance setup and integrator
// (dsinit, dspace) of the reference code, for orbits of 225 minutes or more.

import "math"

// dpper applies the lunar-solar long-period periodics. By design they are
// zero at epoch. With init set it only evaluates them (the initial call);
// otherwise it applies them to the elements, using the Lyddane
// modification below 0.2 rad of inclination.
func dpper(r *satrec, t float64, init bool, ep, inclp, nodep, argpp, mp float64) (float64, float64, float64, float64, float64) {
	const (
		zns = 1.19459e-5
		zes = 0.01675
		znl = 1.5835218e-4
		zel = 0.05490
	)

	// Time-varying periodics. The initial call must use time zero.
	zm := r.zmos + zns*t
	if init {
		zm = r.zmos
	}
	zf := zm + 2.0*zes*math.Sin(zm)
	sinzf := math.Sin(zf)
	f2 := 0.5*sinzf*sinzf - 0.25
	f3 := -0.5 * sinzf * math.Cos(zf)
	ses := r.se2*f2 + r.se3*f3
	sis := r.si2*f2 + r.si3*f3
	sls := r.sl2*f2 + r.sl3*f3 + r.sl4*sinzf
	sghs := r.sgh2*f2 + r.sgh3*f3 + r.sgh4*sinzf
	shs := r.sh2*f2 + r.sh3*f3
	zm = r.zmol + znl*t
	if init {
		zm = r.zmol
	}
	zf = zm + 2.0*zel*math.Sin(zm)
	sinzf = math.Sin(zf)
	f2 = 0.5*sinzf*sinzf - 0.25
	f3 = -0.5 * sinzf * math.Cos(zf)
	sel := r.ee2*f2 + r.e3*f3
	sil := r.xi2*f2 + r.xi3*f3
	sll := r.xl2*f2 + r.xl3*f3 + r.xl4*sinzf
	sghl := r.xgh2*f2 + r.xgh3*f3 + r.xgh4*sinzf
	shll := r.xh2*f2 + r.xh3*f3
	pe := ses + sel
	pinc := sis + sil
	pl := sls + sll
	pgh := sghs + sghl
	ph := shs + shll

	if !init {
		pe -= r.peo
		pinc -= r.pinco
		pl -= r.plo
		pgh -= r.pgho
		ph -= r.pho
		inclp += pinc
		ep += pe
		sinip := math.Sin(inclp)
		cosip := math.Cos(inclp)

		// The GSFC form tests the perturbed inclination here; the
		// original STR#3 tested the epoch inclination.
		if inclp >= 0.2 {
			ph /= sinip
			pgh -= cosip * ph
			argpp += pgh
			nodep += ph
			mp += pl
		} else {
			// Lyddane modification.
			sinop := math.Sin(nodep)
			cosop := math.Cos(nodep)
			alfdp := sinip * sinop
			betdp := sinip * cosop
			dalf := ph*cosop + pinc*cosip*sinop
			dbet := -ph*sinop + pinc*cosip*cosop
			alfdp += dalf
			betdp += dbet
			nodep = math.Mod(nodep, twoPi)
			xls := mp + argpp + pl + pgh + (cosip-pinc*sinip)*nodep
			xnoh := nodep
			nodep = math.Atan2(alfdp, betdp)
			if math.Abs(xnoh-nodep) > math.Pi {
				if nodep < xnoh {
					nodep += twoPi
				} else {
					nodep -= twoPi
				}
			}
			mp += pl
			argpp = xls - mp - cosip*nodep
		}
	}
	return ep, inclp, nodep, argpp, mp
}

// dscomOut carries the dscom quantities that dsinit needs. The lunar-solar
// coefficients that dpper uses later go straight into the satrec.
type dscomOut struct {
	sinim, cosim, emsq, em, nm           float64
	s1, s2, s3, s4, s5                   float64
	ss1, ss2, ss3, ss4, ss5              float64
	sz1, sz3, sz11, sz13, sz21, sz23     float64
	sz31, sz33                           float64
	z1, z3, z11, z13, z21, z23, z31, z33 float64
}

// dscom computes the deep-space terms shared by the secular and periodic
// parts: solar and lunar geometry at the epoch.
func dscom(r *satrec, epoch, ep, argpp, tc, inclp, nodep, np float64) dscomOut {
	const (
		zes    = 0.01675
		zel    = 0.05490
		c1ss   = 2.9864797e-6
		c1l    = 4.7968065e-7
		zsinis = 0.39785416
		zcosis = 0.91744867
		zcosgs = 0.1945905
		zsings = -0.98088458
	)

	var o dscomOut
	nm := np
	em := ep
	snodm := math.Sin(nodep)
	cnodm := math.Cos(nodep)
	sinomm := math.Sin(argpp)
	cosomm := math.Cos(argpp)
	sinim := math.Sin(inclp)
	cosim := math.Cos(inclp)
	emsq := em * em
	betasq := 1.0 - emsq
	rtemsq := math.Sqrt(betasq)

	// Lunar-solar terms.
	r.peo = 0.0
	r.pinco = 0.0
	r.plo = 0.0
	r.pgho = 0.0
	r.pho = 0.0
	day := epoch + 18261.5 + tc/1440.0
	xnodce := math.Mod(4.5236020-9.2422029e-4*day, twoPi)
	stem := math.Sin(xnodce)
	ctem := math.Cos(xnodce)
	zcosil := 0.91375164 - 0.03568096*ctem
	zsinil := math.Sqrt(1.0 - zcosil*zcosil)
	zsinhl := 0.089683511 * stem / zsinil
	zcoshl := math.Sqrt(1.0 - zsinhl*zsinhl)
	gam := 5.8351514 + 0.0019443680*day
	zx := 0.39785416 * stem / zsinil
	zy := zcoshl*ctem + 0.91744867*zsinhl*stem
	zx = math.Atan2(zx, zy)
	zx = gam + zx - xnodce
	zcosgl := math.Cos(zx)
	zsingl := math.Sin(zx)

	// Solar terms first, then lunar on the second pass.
	zcosg := zcosgs
	zsing := zsings
	zcosi := zcosis
	zsini := zsinis
	zcosh := cnodm
	zsinh := snodm
	cc := c1ss
	xnoi := 1.0 / nm

	var s1, s2, s3, s4, s5, s6, s7 float64
	var ss1, ss2, ss3, ss4, ss5, ss6, ss7 float64
	var z1, z2, z3, z11, z12, z13, z21, z22, z23, z31, z32, z33 float64
	var sz1, sz2, sz3, sz11, sz12, sz13, sz21, sz22, sz23, sz31, sz32, sz33 float64

	for lsflg := 1; lsflg <= 2; lsflg++ {
		a1 := zcosg*zcosh + zsing*zcosi*zsinh
		a3 := -zsing*zcosh + zcosg*zcosi*zsinh
		a7 := -zcosg*zsinh + zsing*zcosi*zcosh
		a8 := zsing * zsini
		a9 := zsing*zsinh + zcosg*zcosi*zcosh
		a10 := zcosg * zsini
		a2 := cosim*a7 + sinim*a8
		a4 := cosim*a9 + sinim*a10
		a5 := -sinim*a7 + cosim*a8
		a6 := -sinim*a9 + cosim*a10

		x1 := a1*cosomm + a2*sinomm
		x2 := a3*cosomm + a4*sinomm
		x3 := -a1*sinomm + a2*cosomm
		x4 := -a3*sinomm + a4*cosomm
		x5 := a5 * sinomm
		x6 := a6 * sinomm
		x7 := a5 * cosomm
		x8 := a6 * cosomm

		z31 = 12.0*x1*x1 - 3.0*x3*x3
		z32 = 24.0*x1*x2 - 6.0*x3*x4
		z33 = 12.0*x2*x2 - 3.0*x4*x4
		z1 = 3.0*(a1*a1+a2*a2) + z31*emsq
		z2 = 6.0*(a1*a3+a2*a4) + z32*emsq
		z3 = 3.0*(a3*a3+a4*a4) + z33*emsq
		z11 = -6.0*a1*a5 + emsq*(-24.0*x1*x7-6.0*x3*x5)
		z12 = -6.0*(a1*a6+a3*a5) + emsq*
			(-24.0*(x2*x7+x1*x8)-6.0*(x3*x6+x4*x5))
		z13 = -6.0*a3*a6 + emsq*(-24.0*x2*x8-6.0*x4*x6)
		z21 = 6.0*a2*a5 + emsq*(24.0*x1*x5-6.0*x3*x7)
		z22 = 6.0*(a4*a5+a2*a6) + emsq*
			(24.0*(x2*x5+x1*x6)-6.0*(x4*x7+x3*x8))
		z23 = 6.0*a4*a6 + emsq*(24.0*x2*x6-6.0*x4*x8)
		z1 = z1 + z1 + betasq*z31
		z2 = z2 + z2 + betasq*z32
		z3 = z3 + z3 + betasq*z33
		s3 = cc * xnoi
		s2 = -0.5 * s3 / rtemsq
		s4 = s3 * rtemsq
		s1 = -15.0 * em * s4
		s5 = x1*x3 + x2*x4
		s6 = x2*x3 + x1*x4
		s7 = x2*x4 - x1*x3

		if lsflg == 1 {
			ss1 = s1
			ss2 = s2
			ss3 = s3
			ss4 = s4
			ss5 = s5
			ss6 = s6
			ss7 = s7
			sz1 = z1
			sz2 = z2
			sz3 = z3
			sz11 = z11
			sz12 = z12
			sz13 = z13
			sz21 = z21
			sz22 = z22
			sz23 = z23
			sz31 = z31
			sz32 = z32
			sz33 = z33
			zcosg = zcosgl
			zsing = zsingl
			zcosi = zcosil
			zsini = zsinil
			zcosh = zcoshl*cnodm + zsinhl*snodm
			zsinh = snodm*zcoshl - cnodm*zsinhl
			cc = c1l
		}
	}

	r.zmol = math.Mod(4.7199672+0.22997150*day-gam, twoPi)
	r.zmos = math.Mod(6.2565837+0.017201977*day, twoPi)

	// Solar coefficients.
	r.se2 = 2.0 * ss1 * ss6
	r.se3 = 2.0 * ss1 * ss7
	r.si2 = 2.0 * ss2 * sz12
	r.si3 = 2.0 * ss2 * (sz13 - sz11)
	r.sl2 = -2.0 * ss3 * sz2
	r.sl3 = -2.0 * ss3 * (sz3 - sz1)
	r.sl4 = -2.0 * ss3 * (-21.0 - 9.0*emsq) * zes
	r.sgh2 = 2.0 * ss4 * sz32
	r.sgh3 = 2.0 * ss4 * (sz33 - sz31)
	r.sgh4 = -18.0 * ss4 * zes
	r.sh2 = -2.0 * ss2 * sz22
	r.sh3 = -2.0 * ss2 * (sz23 - sz21)

	// Lunar coefficients.
	r.ee2 = 2.0 * s1 * s6
	r.e3 = 2.0 * s1 * s7
	r.xi2 = 2.0 * s2 * z12
	r.xi3 = 2.0 * s2 * (z13 - z11)
	r.xl2 = -2.0 * s3 * z2
	r.xl3 = -2.0 * s3 * (z3 - z1)
	r.xl4 = -2.0 * s3 * (-21.0 - 9.0*emsq) * zel
	r.xgh2 = 2.0 * s4 * z32
	r.xgh3 = 2.0 * s4 * (z33 - z31)
	r.xgh4 = -18.0 * s4 * zel
	r.xh2 = -2.0 * s2 * z22
	r.xh3 = -2.0 * s2 * (z23 - z21)

	o.sinim, o.cosim, o.emsq, o.em, o.nm = sinim, cosim, emsq, em, nm
	o.s1, o.s2, o.s3, o.s4, o.s5 = s1, s2, s3, s4, s5
	o.ss1, o.ss2, o.ss3, o.ss4, o.ss5 = ss1, ss2, ss3, ss4, ss5
	o.sz1, o.sz3, o.sz11, o.sz13, o.sz21, o.sz23 = sz1, sz3, sz11, sz13, sz21, sz23
	o.sz31, o.sz33 = sz31, sz33
	o.z1, o.z3, o.z11, o.z13, o.z21, o.z23, o.z31, o.z33 = z1, z3, z11, z13, z21, z23, z31, z33
	return o
}

// dsinit sets up the deep-space secular rates and, for half-day and
// one-day orbits, the geopotential resonance terms.
func dsinit(r *satrec, d dscomOut, t, tc, xpidot, eccsq, inclm float64) {
	const (
		q22    = 1.7891679e-6
		q31    = 2.1460748e-6
		q33    = 2.2123015e-7
		root22 = 1.7891679e-6
		root44 = 7.3636953e-9
		root54 = 2.1765803e-9
		rptim  = 4.37526908801129966e-3 // 7.29211514668855e-5 rad/s
		root32 = 3.7393792e-7
		root52 = 1.1428639e-7
		znl    = 1.5835218e-4
		zns    = 1.19459e-5
	)

	cosim, sinim, emsq := d.cosim, d.sinim, d.emsq
	em, nm := d.em, d.nm

	r.irez = 0
	if 0.0034906585 < nm && nm < 0.0052359877 {
		r.irez = 1
	}
	if 8.26e-3 <= nm && nm <= 9.24e-3 && em >= 0.5 {
		r.irez = 2
	}

	// Solar terms.
	ses := d.ss1 * zns * d.ss5
	sis := d.ss2 * zns * (d.sz11 + d.sz13)
	sls := -zns * d.ss3 * (d.sz1 + d.sz3 - 14.0 - 6.0*emsq)
	sghs := d.ss4 * zns * (d.sz31 + d.sz33 - 6.0)
	shs := -zns * d.ss2 * (d.sz21 + d.sz23)
	// Guard for inclinations near 0 and 180 degrees.
	if inclm < 5.2359877e-2 || inclm > math.Pi-5.2359877e-2 {
		shs = 0.0
	}
	if sinim != 0.0 {
		shs = shs / sinim
	}
	sgs := sghs - cosim*shs

	// Lunar terms.
	r.dedt = ses + d.s1*znl*d.s5
	r.didt = sis + d.s2*znl*(d.z11+d.z13)
	r.dmdt = sls - znl*d.s3*(d.z1+d.z3-14.0-6.0*emsq)
	sghl := d.s4 * znl * (d.z31 + d.z33 - 6.0)
	shll := -znl * d.s2 * (d.z21 + d.z23)
	if inclm < 5.2359877e-2 || inclm > math.Pi-5.2359877e-2 {
		shll = 0.0
	}
	r.domdt = sgs + sghl
	r.dnodt = shs
	if sinim != 0.0 {
		r.domdt = r.domdt - cosim/sinim*shll
		r.dnodt = r.dnodt + shll/sinim
	}

	// Resonance effects. The reference also applies the secular updates of
	// em, inclm and the rest with t here; they are zero because dsinit only
	// runs at epoch, and nothing below reads them, so they are left out.
	theta := math.Mod(r.gsto+tc*rptim, twoPi)

	if r.irez != 0 {
		aonv := math.Pow(nm/wgs72Xke, x2o3)

		// Geopotential resonance for 12-hour orbits.
		if r.irez == 2 {
			cosisq := cosim * cosim
			// The reference saves em and restores it afterwards; nothing
			// reads it after this block, so only emsq is restored.
			em = r.ecco
			emsqo := emsq
			emsq = eccsq
			eoc := em * emsq
			g201 := -0.306 - (em-0.64)*0.440

			var g211, g310, g322, g410, g422, g520, g521, g532, g533 float64
			if em <= 0.65 {
				g211 = 3.616 - 13.2470*em + 16.2900*emsq
				g310 = -19.302 + 117.3900*em - 228.4190*emsq + 156.5910*eoc
				g322 = -18.9068 + 109.7927*em - 214.6334*emsq + 146.5816*eoc
				g410 = -41.122 + 242.6940*em - 471.0940*emsq + 313.9530*eoc
				g422 = -146.407 + 841.8800*em - 1629.014*emsq + 1083.4350*eoc
				g520 = -532.114 + 3017.977*em - 5740.032*emsq + 3708.2760*eoc
			} else {
				g211 = -72.099 + 331.819*em - 508.738*emsq + 266.724*eoc
				g310 = -346.844 + 1582.851*em - 2415.925*emsq + 1246.113*eoc
				g322 = -342.585 + 1554.908*em - 2366.899*emsq + 1215.972*eoc
				g410 = -1052.797 + 4758.686*em - 7193.992*emsq + 3651.957*eoc
				g422 = -3581.690 + 16178.110*em - 24462.770*emsq + 12422.520*eoc
				if em > 0.715 {
					g520 = -5149.66 + 29936.92*em - 54087.36*emsq + 31324.56*eoc
				} else {
					g520 = 1464.74 - 4664.75*em + 3763.64*emsq
				}
			}
			if em < 0.7 {
				g533 = -919.22770 + 4988.6100*em - 9064.7700*emsq + 5542.21*eoc
				g521 = -822.71072 + 4568.6173*em - 8491.4146*emsq + 5337.524*eoc
				g532 = -853.66600 + 4690.2500*em - 8624.7700*emsq + 5341.4*eoc
			} else {
				g533 = -37995.780 + 161616.52*em - 229838.20*emsq + 109377.94*eoc
				g521 = -51752.104 + 218913.95*em - 309468.16*emsq + 146349.42*eoc
				g532 = -40023.880 + 170470.89*em - 242699.48*emsq + 115605.82*eoc
			}

			sini2 := sinim * sinim
			f220 := 0.75 * (1.0 + 2.0*cosim + cosisq)
			f221 := 1.5 * sini2
			f321 := 1.875 * sinim * (1.0 - 2.0*cosim - 3.0*cosisq)
			f322 := -1.875 * sinim * (1.0 + 2.0*cosim - 3.0*cosisq)
			f441 := 35.0 * sini2 * f220
			f442 := 39.3750 * sini2 * sini2
			f522 := 9.84375 * sinim * (sini2*(1.0-2.0*cosim-5.0*cosisq) +
				0.33333333*(-2.0+4.0*cosim+6.0*cosisq))
			f523 := sinim * (4.92187512*sini2*(-2.0-4.0*cosim+
				10.0*cosisq) + 6.56250012*(1.0+2.0*cosim-3.0*cosisq))
			f542 := 29.53125 * sinim * (2.0 - 8.0*cosim + cosisq*
				(-12.0+8.0*cosim+10.0*cosisq))
			f543 := 29.53125 * sinim * (-2.0 - 8.0*cosim + cosisq*
				(12.0+8.0*cosim-10.0*cosisq))
			xno2 := nm * nm
			ainv2 := aonv * aonv
			temp1 := 3.0 * xno2 * ainv2
			temp := temp1 * root22
			r.d2201 = temp * f220 * g201
			r.d2211 = temp * f221 * g211
			temp1 = temp1 * aonv
			temp = temp1 * root32
			r.d3210 = temp * f321 * g310
			r.d3222 = temp * f322 * g322
			temp1 = temp1 * aonv
			temp = 2.0 * temp1 * root44
			r.d4410 = temp * f441 * g410
			r.d4422 = temp * f442 * g422
			temp1 = temp1 * aonv
			temp = temp1 * root52
			r.d5220 = temp * f522 * g520
			r.d5232 = temp * f523 * g532
			temp = 2.0 * temp1 * root54
			r.d5421 = temp * f542 * g521
			r.d5433 = temp * f543 * g533
			r.xlamo = math.Mod(r.mo+r.nodeo+r.nodeo-theta-theta, twoPi)
			r.xfact = r.mdot + r.dmdt + 2.0*(r.nodedot+r.dnodt-rptim) - r.noUnkozai
			emsq = emsqo
		}

		// Synchronous resonance terms.
		if r.irez == 1 {
			g200 := 1.0 + emsq*(-2.5+0.8125*emsq)
			g310 := 1.0 + 2.0*emsq
			g300 := 1.0 + emsq*(-6.0+6.60937*emsq)
			f220 := 0.75 * (1.0 + cosim) * (1.0 + cosim)
			f311 := 0.9375*sinim*sinim*(1.0+3.0*cosim) - 0.75*(1.0+cosim)
			f330 := 1.0 + cosim
			f330 = 1.875 * f330 * f330 * f330
			r.del1 = 3.0 * nm * nm * aonv * aonv
			r.del2 = 2.0 * r.del1 * f220 * g200 * q22
			r.del3 = 3.0 * r.del1 * f330 * g300 * q33 * aonv
			r.del1 = r.del1 * f311 * g310 * q31 * aonv
			r.xlamo = math.Mod(r.mo+r.nodeo+r.argpo-theta, twoPi)
			r.xfact = r.mdot + xpidot - rptim + r.dmdt + r.domdt + r.dnodt - r.noUnkozai
		}

		// Starting state for the integrator.
		r.xli = r.xlamo
		r.xni = r.noUnkozai
	}
}

// dspace adds the deep-space secular effects and integrates the resonance
// terms (Euler-Maclaurin, 720-minute steps) from epoch to t.
func dspace(r *satrec, t, tc, em, argpm, inclm, mm, nodem, nm float64) (float64, float64, float64, float64, float64, float64) {
	const (
		fasx2 = 0.13130908
		fasx4 = 2.8843198
		fasx6 = 0.37448087
		g22   = 5.7686396
		g32   = 0.95240898
		g44   = 1.8014998
		g52   = 1.0508330
		g54   = 4.4108898
		rptim = 4.37526908801129966e-3 // 7.29211514668855e-5 rad/s
		stepp = 720.0
		stepn = -720.0
		step2 = 259200.0
	)

	theta := math.Mod(r.gsto+tc*rptim, twoPi)
	em = em + r.dedt*t
	inclm = inclm + r.didt*t
	argpm = argpm + r.domdt*t
	nodem = nodem + r.dnodt*t
	mm = mm + r.dmdt*t

	if r.irez == 0 {
		return em, argpm, inclm, mm, nodem, nm
	}

	// The reference caches atime, xli and xni between calls; starting
	// from epoch every time gives the same answer.
	atime := 0.0
	xni := r.noUnkozai
	xli := r.xlamo

	delt := stepn
	if t > 0.0 {
		delt = stepp
	}

	var xndt, xldot, xnddt, ft float64
	for {
		if r.irez != 2 {
			// Near-synchronous resonance terms.
			xndt = r.del1*math.Sin(xli-fasx2) + r.del2*math.Sin(2.0*(xli-fasx4)) +
				r.del3*math.Sin(3.0*(xli-fasx6))
			xldot = xni + r.xfact
			xnddt = r.del1*math.Cos(xli-fasx2) +
				2.0*r.del2*math.Cos(2.0*(xli-fasx4)) +
				3.0*r.del3*math.Cos(3.0*(xli-fasx6))
			xnddt = xnddt * xldot
		} else {
			// Near half-day resonance terms.
			xomi := r.argpo + r.argpdot*atime
			x2omi := xomi + xomi
			x2li := xli + xli
			xndt = r.d2201*math.Sin(x2omi+xli-g22) + r.d2211*math.Sin(xli-g22) +
				r.d3210*math.Sin(xomi+xli-g32) + r.d3222*math.Sin(-xomi+xli-g32) +
				r.d4410*math.Sin(x2omi+x2li-g44) + r.d4422*math.Sin(x2li-g44) +
				r.d5220*math.Sin(xomi+xli-g52) + r.d5232*math.Sin(-xomi+xli-g52) +
				r.d5421*math.Sin(xomi+x2li-g54) + r.d5433*math.Sin(-xomi+x2li-g54)
			xldot = xni + r.xfact
			xnddt = r.d2201*math.Cos(x2omi+xli-g22) + r.d2211*math.Cos(xli-g22) +
				r.d3210*math.Cos(xomi+xli-g32) + r.d3222*math.Cos(-xomi+xli-g32) +
				r.d5220*math.Cos(xomi+xli-g52) + r.d5232*math.Cos(-xomi+xli-g52) +
				2.0*(r.d4410*math.Cos(x2omi+x2li-g44)+
					r.d4422*math.Cos(x2li-g44)+r.d5421*math.Cos(xomi+x2li-g54)+
					r.d5433*math.Cos(-xomi+x2li-g54))
			xnddt = xnddt * xldot
		}

		if math.Abs(t-atime) < stepp {
			ft = t - atime
			break
		}
		xli = xli + xldot*delt + xndt*step2
		xni = xni + xndt*delt + xnddt*step2
		atime = atime + delt
	}

	nm = xni + xndt*ft + xnddt*ft*ft*0.5
	xl := xli + xldot*ft + xndt*ft*ft*0.5
	var dndt float64
	if r.irez != 1 {
		mm = xl - 2.0*nodem + 2.0*theta
		dndt = nm - r.noUnkozai
	} else {
		mm = xl - nodem - argpm + theta
		dndt = nm - r.noUnkozai
	}
	nm = r.noUnkozai + dndt
	return em, argpm, inclm, mm, nodem, nm
}
