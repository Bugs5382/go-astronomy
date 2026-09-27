"""Reference ISS positions from python-sgp4 for the Go cross-check test.

Run with
    python satellite/testdata/iss_crosscheck.py > satellite/testdata/iss_python.json

with python-sgp4 installed (2.25 produced the committed file). The element set is
synthetic: an ISS-like orbit with an epoch of 2026-09-26, used only to compare the
two propagators.
"""
import json
import sgp4
from sgp4.api import Satrec, jday, accelerated
from sgp4.propagation import gstime

# Checksum digits corrected (the originals were 3 and 5; the lines tally to 0 and 8).
L1 = '1 25544U 98067A   26269.51782528  .00016717  00000-0  30306-3 0  9990'
L2 = '2 25544  51.6416 247.4627 0006703 130.5360 325.0288 15.49450470 12348'

INSTANTS = [
    (2026, 9, 26, 12, 25, 40.104192),  # the epoch
    (2026, 9, 26, 13, 25, 40.104192),
    (2026, 9, 27, 0, 0, 0.0),
    (2026, 9, 25, 12, 0, 0.0),
    (2026, 9, 29, 18, 45, 30.25),
    (2026, 10, 1, 6, 30, 15.5),
    (2026, 9, 1, 0, 0, 0.0),
]

sat = Satrec.twoline2rv(L1, L2)
out = {'sgp4_version': sgp4.__version__, 'accelerated': accelerated,
       'line1': L1, 'line2': L2, 'cases': [], 'gmst': []}
for y, mo, d, h, mi, s in INSTANTS:
    jd, fr = jday(y, mo, d, h, mi, s)
    e, r, v = sat.sgp4(jd, fr)
    out['cases'].append({'utc': [y, mo, d, h, mi, s], 'error': e,
                         'r': list(r), 'v': list(v)})
    out['gmst'].append({'jd': jd + fr, 'gmst': gstime(jd + fr)})
print(json.dumps(out, indent=1))
