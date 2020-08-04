import os

for root, dirs, files in os.walk(r'D:\PixivDownload'):
    print(root, dirs, files)
    break
