import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns

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


def calcAverageTime(publish, received, parameter):
	publish = publish[['_time', 'messageID', 'topic']]
	received = received[['_time', 'messageID', 'topic']]

	joined = publish.merge(received, on=['messageID', 'topic'])
	joined['diff'] = ((joined['_time_y'] - joined['_time_x'])/ pd.Timedelta(microseconds=1)).astype(int)
	# joined.head(10)

	#Make the db in memory
	conn = sqlite3.connect(':memory:')
	#write the tables
	joined.to_sql('joined', conn, index=False)
	expTime.to_sql('expTime', conn, index=False)

	qry = '''
	    select  
	        joined._time_x,
	        joined.diff,
	        joined.messageID,
	        joined.topic,
	        expTime.experiment,
	        expTime.'''parameter'''
	    from
	        joined join expTime on
	        joined._time_x between expTime.start and expTime.end
	    '''
	dfNew = pd.read_sql_query(qry, conn)
	dfNew = dfNew.set_index('experiment').rename(columns={"_time_x": "_time"})#.drop(columns=["messageID"])
	# dfNew.head(20)

	#Average propagation time per interval
	df = dfNew.drop(columns=['_time', 'messageID', 'topic'])

	avgPropExp = df.groupby(['experiment']).agg('mean')
	avgPropExp.reset_index(inplace=True)
	avgPropExp = avgPropExp.drop(columns=['experiment'])

	avgProp = avgPropExp.groupby(['interval']).agg({'diff':['mean','std']})
	avgProp.columns = avgProp.columns.droplevel(0)
	avgProp.reset_index(inplace=True)

	# avgProp.head(100)

def plotPropTime(df, topology, parameter, parameter_name):

	y_labels = df['interval'].astype(str).to_numpy()
	# print(y_labels)

	plt.style.use('ggplot')
	# kwargs = dict(color=['hotpink'], alpha=0.9)#, density=True)

	fig, ax = plt.subplots(constrained_layout=False)
	ax.grid(alpha=0.4)

	fig.tight_layout()
	fig.subplots_adjust(left=0.19, bottom=0.09, right=0.98, top=0.92, wspace=0.17, hspace=0.17)

	plt.rcParams.update({'figure.figsize':(7,5), 'figure.dpi':100})

	plt.gca().set(title='Propagation time by '+parameter_name, ylabel='parameter', xlabel="Time [s]")

	plt.rcParams.update({'font.size': 14})
	# plt.hist([x1, x2, x6], **kwargs, label=['GS 1 topic', 'Vanilla', 'Squelching'])
	# plt.legend(loc='upper center', bbox_to_anchor=(0.58, 1))
	# plt.gca().xaxis.set_major_locator(ticker.MultipleLocator(2))

	plt.xticks(fontsize=18)
	plt.yticks(fontsize=16)

	fig.set_size_inches(11, 4.3)

	#Set y axis labels
	ax.barh(y_labels, df['mean'], xerr=df['std'], align='center')
	ax.set_yticks(y_labels, labels=parameter_name)
	ax.invert_yaxis()  # labels read top-to-bottom
	ax.set_xlabel('Propagation time [μs]')

	fig.savefig('./figures/propTime_'+topology+'_'+parameter+'.pdf', format='pdf', facecolor='white', edgecolor='none', bbox_inches='tight', dpi=600)
    fig.savefig('./figures/propTime_'+topology+'_'+parameter+'.png', format='png', facecolor='white', edgecolor='none', bbox_inches='tight', dpi=600)


	plt.show()

	
#############################################
#	Read config file and load data into the variables
############################################# 
filepath = '/root/flexi-pipe/config.go'
# open the file and read through it line by line
with open(filepath, 'r') as file_object:
    line = file_object.readline()
    while line:
        # at each line check for a match with a regex
        key, match = functions._parse_line(line)

        if key == 'token':
            token = match.group('token')
        elif key == 'url':
            url = match.group('url')
        elif key == 'org':
            org = match.group('org')
        elif key == 'bucket':
            bucket = match.group('bucket')
        
        line = file_object.readline()
# url="http://192.168.20.58:8086"
url = "http://localhost:8086"

#############################################
#	Global start and end time
#############################################
start_time = 1691423149
end_time = 1691473008

#############################################
#	Graphs to generate
#############################################
#Need to get:
#	measurement 		(influx measurement)
#	measurement_name 	(for printing on the graph header)
#	topology 				(experiment type - unl, general, validator)
#	parameter			(gossipsub parameter, d, dlo, dhigh....)
#	parameter_name		(for printing on the graph header)
#   grouping_key        (key used for grouping and counting)

#############################################
#	Query experiments file and influx to get all the data
#############################################
experiments = functions.experiment(start_time, end_time, './experiments.csv')
print(experiments)

#############################################
#	Loop to procces each graph
#############################################
# oldMeasurement = ''
with open('graphs_parameters.csv', 'r') as file:
    reader = csv.DictReader(file)
    data = list(reader)
    for graph in data:
        print(graph)
        print("Influx query")
        published 	= functions.from_influx(url, token, org, "publishMessage", graph['start'], graph['end'], graph['grouping_key'])
        received 	= functions.from_influx(url, token, org, "deliverMessage", graph['start'], graph['end'], graph['grouping_key'])

        exp = experiments.loc[experiments['topology'] == graph['topology']]
        df = calcAverageTime(published, received, graph['parameter'])
        print("Data tratead")

        generate_graph(graph['measurement'], graph['measurement_name'], graph['topology'], gb, graph['parameter'],graph['parameter_name'])

        print("Graph generated")
