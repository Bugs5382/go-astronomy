"""Reference TEME states for the OMM samples, from python-sgp4's own OMM reader.

Run with python-sgp4 installed:

    python satellite/testdata/omm_python.py > satellite/testdata/omm_python.txt

The two samples (omm-vanguard.json and omm-vanguard.xml) are the ones python-sgp4
ships as sample_omm.json and sample_omm.xml.
"""
import os
from sgp4 import omm, __version__
from sgp4.api import Satrec

here = os.path.dirname(os.path.abspath(__file__))
print('# python-sgp4 %s: TEME position (km) and velocity (km/s) for the OMM samples' % __version__)
print('# Rows: <file> <minutes since epoch> x y z vx vy vz')
for name, parse in (('omm-vanguard.json', omm.parse_xml), ('omm-vanguard.xml', omm.parse_xml)):
    path = os.path.join(here, name)
    with open(path) as f:
        if name.endswith('.json'):
            import json
            records = json.load(f)
        else:
            records = list(parse(f))
    for fields in records:
        fields = {k: str(v) for k, v in fields.items()}
        sat = Satrec()
        omm.initialize(sat, fields)
        for m in (0.0, 60.0, 1440.0, -720.0):
            e, r, v = sat.sgp4_tsince(m)
            assert e == 0, e
            print('%s %.1f %.9f %.9f %.9f %.12f %.12f %.12f' % ((name, m) + tuple(r) + tuple(v)))
