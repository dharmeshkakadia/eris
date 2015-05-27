import os
from subprocess import call
import time

def replace_all(path='.', old="github.com/ethereum", new="github.com/project-douglas"):
    in_this_dir = os.listdir(path)
    for f in in_this_dir:
        f = os.path.join(path, f)
        if os.path.isdir(f):
            replace_all(path=f, old=old, new=new)
        elif f[-3:] == ".go":
            d = open(f)
            src = d.readlines()
            d.close()
            d = open(f, "w")
            for s in src:
                s = s.replace(old, new)
                d.write(s)
            d.close()

mods = ["ethchain", "ethstate", "ethvm", "ethwire", "ethtrie", "ethutil", "ethdb", "ethlog", "ethminer", "ethcrypto", "ethpipe", "ethreact", "ethrpc", "ethtest"]


#for m in mods:
    #replace_all(old="github.com/eris-ltd/thelonious/"+m, new="github.com/eris-ltd/thelonious/monk"+m[3:])
    #replace_all(old=m, new="monk"+m[3:])

replace_all(old="EthManager", new="NodeManager")

