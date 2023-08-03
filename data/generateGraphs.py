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

import functions


#############################################
#	Read config file and load data into the variables
#############################################
rx_dict = {
    'token': re.compile(r'var token = "(?P<token>.*)"\n'),
    'url': re.compile(r'var url = "(?P<url>.*)"\n'),
    'org': re.compile(r'var org = "(?P<org>.*)"\n'),
    'bucket': re.compile(r'var bucket = "(?P<bucket>.*)"\n'),
}
    
filepath = '/root/flexi-pipe/config.go'
# open the file and read through it line by line
with open(filepath, 'r') as file_object:
    line = file_object.readline()
    while line:
        # at each line check for a match with a regex
        key, match = _parse_line(line)

        if key == 'token':
            token = match.group('token')
        elif key == 'url':
            url = match.group('url')
        elif key == 'org':
            org = match.group('org')
        elif key == 'bucket':
            bucket = match.group('bucket')
        
        line = file_object.readline()
url="http://192.168.20.58:8086"

#############################################
#	Global start and end time
#############################################

#############################################
#	Graphs to generate
#############################################
#Need to get:
#	measurement 		(influx measurement)
#	measurement_name 	(for printing on the graph header)
#	topology 				(experiment type - unl, general, validator)
#	parameter			(gossipsub parameter, d, dlo, dhigh....)
#	parameter_name		(for printing on the graph header)

measurement = "duplicateMessage"
measurement_name = "Message Overhead"
topology = "unl"
parameter = "interval"
parameter_name = "interval"


#############################################
#	Query experiments file and influx to get all the data
#############################################
experiments = experiment(start_time, end_time, './experiments.csv')
traces = from_influx(url, token, org, measurement, start_time, end_time)

#############################################
#	Loop to procces each graph
#############################################

#get topology
exp = experiment.loc[experiment['topology'] == topology]

gb = group_time(traces, exp, parameter)

generate_graph(measurement, topology, gb, parameter)