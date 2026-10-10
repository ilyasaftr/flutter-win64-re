#!/usr/bin/env python3
"""Grok direct write: bypass MCP premium gate by writing Reqable config files + restart.
Usage: python3 grok_direct_write.py rewrite|breakpoint|list [--restart]
"""
import json, os, sys, uuid, shutil, subprocess, time

UID="816e77255b96a7ff"
BASE=os.path.expanduser(f"~/Library/Application Support/com.reqable.macosx/user/{UID}")

def load(name):
    p=f"{BASE}/config/{name}"
    with open(p) as f: return json.load(f), p

def save(cfg, path):
    tmp=path+".tmp"
    with open(tmp,"w") as f: json.dump(cfg,f)
    os.replace(tmp, path)

def add_rewrite(name, url, action_type=2, code=200, body_payload='{"ok": true}', method="", wildcard=True):
    cfg,path=load("capture_rewrite")
    rid=str(uuid.uuid4())
    entry={"id":rid,"name":name,"method":method,"url":url,"wildcard":wildcard,
     "action":{"type":action_type,"code":code,"statusLineEnabled":False,"headersEnabled":False,"bodyEnabled":True,"headers":[],"body":{"type":1,"payload":body_payload}},"isEnabled":True}
    cfg["rewrites"].append(entry)
    save(cfg,path)
    print(f"rewrite {rid} {name} -> restart Reqable to load")
    return rid

def add_rewrite_modify(name, url, pattern, replacement, wildcard=True):
    cfg,path=load("capture_rewrite")
    rid=str(uuid.uuid4())
    entry={"id":rid,"name":name,"method":"","url":url,"wildcard":wildcard,
     "action":{"type":4,"modifies":[{"id":str(uuid.uuid4()),"type":0,"modify":{"pattern":pattern,"replacement":replacement,"regex":True,"caseSensitive":False},"isEnabled":True}],
     "modifyQueryTemplate":None,"modifyHeaderTemplate":None,"modifyBodyTemplate":None},"isEnabled":True}
    cfg["rewrites"].append(entry)
    save(cfg,path)
    print(f"modify rewrite {rid} {name}")
    return rid

def add_breakpoint(name, url, wildcard=True):
    cfg,path=load("capture_breakpoint")
    bid=str(uuid.uuid4())
    entry={"id":bid,"name":name,"method":"","url":url,"wildcard":wildcard,"isRequestEnabled":True,"isResponseEnabled":True,"isEnabled":True}
    cfg["breakpoints"].append(entry)
    save(cfg,path)
    print(f"breakpoint {bid} {name}")
    return bid

def list_all():
    for n in ["capture_rewrite","capture_breakpoint"]:
        cfg,_=load(n)
        key="rewrites" if "rewrites" in cfg else "breakpoints"
        print(f"{n} ({len(cfg[key])}):")
        for e in cfg[key]:
            print(f"  {e['id']} {e['name']} {e['url']}")

if __name__=="__main__":
    cmd=sys.argv[1] if len(sys.argv)>1 else "list"
    if cmd=="list": list_all()
    elif cmd=="rewrite": add_rewrite(sys.argv[2], sys.argv[3], body_payload=sys.argv[4] if len(sys.argv)>4 else '{"ok": true}')
    elif cmd=="rewrite-modify": add_rewrite_modify(sys.argv[2], sys.argv[3], sys.argv[4], sys.argv[5])
    elif cmd=="breakpoint": add_breakpoint(sys.argv[2], sys.argv[3])
    else: print(__doc__)
