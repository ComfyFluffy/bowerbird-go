#%%
import os
import hashlib
import shutil

from glob import glob

d = r'D:\PixivDownload'
dd = r'D:\pixiv'


def hashFile(fn: str) -> str:
    sha1 = hashlib.sha1()
    # print("hash", fn)
    with open(fn, 'rb') as f:
        while True:
            data = f.read(10 * 1024)
            if not data:
                break
        sha1.update(data)
    return sha1.hexdigest()


os.chdir(d)

flog = open('d:/pixiv/log2.log', 'a')


def mvDiff():
    for x in os.listdir("."):
        p = os.path.join(d, x)
        if os.path.isdir(p):
            os.chdir(p)
            for x in glob("*_*_*"):
                x2 = x[:-19] + x[-4:]
                dst = os.path.join(dd, x2)

                if os.path.isfile(x) and os.path.isfile(x2) and hashFile(
                        x2) == hashFile(x):
                    print(p, x2, dst)
                    # if os.path.exists(dst):
                    #     raise FileExistsError
                    shutil.move(x2, dst)
                    # exit()
                else:
                    print(p, x2, dst, file=flog, flush=True)
            for x in glob("*_*"):
                x2 = x[:-15]
                dst = os.path.join(dd, x2)

                if os.path.isdir(x2) and os.path.isdir(x) \
                    and os.listdir(x2) == os.listdir(x):
                    l1 = set()
                    l2 = set()
                    for f1 in os.listdir(x):
                        l1.add(hashFile(os.path.join(x, f1)))
                    for f2 in os.listdir(x2):
                        l2.add(hashFile(os.path.join(x2, f2)))
                    if l1 == l2:
                        print(p, x2, dst)
                        if os.path.exists(dst):
                            raise FileExistsError
                        os.rename(x2, dst)
                    else:
                        print(p, x2, dst, file=flog, flush=True)


def checkHash():
    os.chdir(dd)
    s1 = set()
    s2 = set()
    for root, dirs, files in os.walk("."):
        for f in files:
            s1.add(hashFile(os.path.join(root, f)))

    os.chdir(d)
    for root, dirs, files in os.walk("."):
        for f in files:
            s2.add(hashFile(os.path.join(root, f)))
    print(s1.issubset(s2), s1.difference(s2))


checkHash()
