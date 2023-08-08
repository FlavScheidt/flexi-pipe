import pandas as pd
import numpy as np

from influxdb_client import InfluxDBClient, Point, Dialect

import re
import time
import datetime

import warnings
from influxdb_client.client.warnings import MissingPivotFunction

import matplotlib.pyplot as plt
import matplotlib.ticker as ticker
import matplotlib.colors as colors

import pandasql as ps
import sqlite3

import csv

# import functions


# Function to parse configurations from go config file
def _parse_line(line):

    rx_dict = {
    'token': re.compile(r'var token = "(?P<token>.*)"\n'),
    'url': re.compile(r'var url = "(?P<url>.*)"\n'),
    'org': re.compile(r'var org = "(?P<org>.*)"\n'),
    'bucket': re.compile(r'var bucket = "(?P<bucket>.*)"\n'),
    }   

    """
    Do a regex search against all defined regexes and
    return the key and match result of the first matching regex

    """
    for key, rx in rx_dict.items():
        match = rx.search(line)
        if match:
            return key, match
    # if there are no matches
    return None, None

    #Get experiment data
def experiment(start_time, end_time, filepath):
    # Retrieve experiments data from csv
    data = pd.read_csv(filepath, header=None)
    df = pd.DataFrame(data)

    #Rename columns
    experiments = df.rename(columns={0: "start", 1: "end", 2: "topology", 3: "runtime", 4: "d", 5: "dlo", 6: "dhi", 7: "dscore", 8: "dlazy", 9: "dout", 10: "gossipFactor", 11: "initialDelay", 12: "interval"}, errors='raise')

    #Correct timestamp
    experiments["start"] = experiments["start"].str.slice(0, 27)
    experiments["end"] = experiments["end"].str.slice(0, 27)

    #String to timestamp
    experiments['startUnix'] = pd.to_datetime(experiments["start"],format="%Y-%m-%d %H:%M:%S.%f").astype('int64') / 10**9
    experiments['endUnix'] = pd.to_datetime(experiments["end"],format="%Y-%m-%d %H:%M:%S.%f").astype('int64') / 10**9

    experiments['startUnix'] = pd.to_timedelta(experiments['startUnix'], unit='s').dt.total_seconds().astype(int)#.astype(str)
    experiments['endUnix'] = pd.to_timedelta(experiments['endUnix'], unit='s').dt.total_seconds().astype(int)#.astype(str)

    #Drop fields we don't mneed for the moment
    exp = experiments.drop(columns=["runtime", "initialDelay"]).sort_values(by=["start"])

    #Get times for different intervals
    # intervals = exp["interval"].drop_duplicates().sort_values().reset_index(drop=True)
    # intervals.head(10)

    expTime = exp[exp['startUnix'].astype(int).between(start_time, end_time)]
    # expTime['experiment'] = expTime.index
    expTime = expTime.reset_index().rename({'index':'experiment'}, axis = 'columns')

    return expTime

#Query from influxdb
def from_influx(url, token, org, measurement, start_time, end_time,grouping_key):
    client = InfluxDBClient(url=url, token=token, org=org,  timeout=900_000)

    # write_api = client.write_api(write_options=SYNCHRONOUS)
    query_api = client.query_api()

    data_frame = query_api.query_data_frame('from(bucket: "gs") '
                                        ' |> range(start: '+str(start_time)+', stop:'+str(end_time)+') '
                                        ' |> filter(fn: (r) => r._measurement == "'+measurement+'") '
                                        ' |> group(columns: ["_measurement", "_field"], mode: "by") '
                                        ' |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")')
    client.close()

    # df = data_frame.drop(columns=['result', 'table','_start', '_stop', '_measurement', 'topic', 'receivedFrom']).sort_values(by=["_time"]).reset_index(drop=True)
    data_frame.reset_index(inplace=True)
    df = data_frame[['_time', grouping_key]].sort_values(by=["_time"]).reset_index(drop=True)
    df["_time"] = pd.to_datetime(df["_time"])

    return df