import pymongo
from pymongo import collection as pymongo_collection

client = pymongo.MongoClient('localhost', ssl=False)
test: pymongo_collection.Collection = client.testgo.test
