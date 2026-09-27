import urllib.parse, urllib.request, datetime, time
out='/Users/shanefroebel/Code/worktrees/Bugs5382/go-astronomy/agent-ab05bb3498a2905e7/satellite/horizons/testdata/'
def q(target, start, stop, step, center="'500@399'", quantities="'2,20'", site=None):
    p={"format":"text","COMMAND":f"'{target}'","OBJ_DATA":"'NO'","MAKE_EPHEM":"'YES'","EPHEM_TYPE":"'OBSERVER'",
       "CENTER":center,"QUANTITIES":quantities,"ANG_FORMAT":"'DEG'","EXTRA_PREC":"'YES'","CSV_FORMAT":"'YES'",
       "TIME_TYPE":"'UT'","START_TIME":f"'{start}'","STOP_TIME":f"'{stop}'","STEP_SIZE":f"'{step}'"}
    if site:
        p["COORD_TYPE"]="'GEODETIC'"; p["SITE_COORD"]=f"'{site}'"
    url="https://ssd.jpl.nasa.gov/api/horizons.api?"+urllib.parse.urlencode(p)
    t=urllib.request.urlopen(url,timeout=120).read().decode()
    time.sleep(1)
    return url,t
def save(name,url,t):
    open(out+name,'w').write(t)
    print(name, len(t), t.count('\n'), 'SOE' if '$$SOE' in t else t[t.find('No ephemeris'):t.find('No ephemeris')+160] )
# window aligned the way the client aligns it: 30-day blocks from the Unix epoch, padded 4 steps each side
W=30*86400; pad=4*3600
t=datetime.datetime(2026,10,1,tzinfo=datetime.timezone.utc).timestamp()
ws=int(t//W)*W
fmt=lambda s: datetime.datetime.fromtimestamp(s,datetime.timezone.utc).strftime('%Y-%m-%d %H:%M')
print("window", fmt(ws-pad), fmt(ws+W+pad))
u,tx=q('-170',fmt(ws-pad),fmt(ws+W+pad),'60 m'); save('jwst-window.txt',u,tx)
u,tx=q('-170','2026-10-01 00:00','2026-10-03 00:00','10 m'); save('jwst-dense.txt',u,tx)
u,tx=q('-211',fmt(ws-pad),fmt(ws+W+pad),'60 m'); save('roman-window-full.txt',u,tx)
u,tx=q('-170','2026-10-01 00:00','2026-10-01 06:00','60 m',center="'coord@399'",quantities="'2,4,20'",site='-0.0005,51.4769,0'); save('jwst-greenwich.txt',u,tx)
u,tx=q('-211',fmt(ws+W-pad),fmt(ws+2*W+pad),'60 m'); save('roman-beyond-coverage.txt',u,tx)
u,tx=q('-211',fmt(ws+W-pad),'2026-10-19 12:00','60 m'); save('roman-clipped.txt',u,tx)
