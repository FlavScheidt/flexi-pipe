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

import functions


def calcBandwidth(message, rpc, expTime, parameter):

	message = message[['_time', '_measurement']].reset_index(drop=True)
	rpc = rpc[['_time', '_measurement']].reset_index(drop=True)

	joined = pd.concat([rpc, message])#on=['receivedFrom', 'topic'])
	joined["_time"] = pd.to_datetime(joined["_time"])

	#Make the db in memory
	conn = sqlite3.connect(':memory:')
	#write the tables
	joined.to_sql('df', conn, index=False)
	expTime.to_sql('expTime', conn, index=False)

	qry = '''
	    select  
	        df._time,
	        df._measurement,
	        expTime.experiment,
	        expTime.'''+parameter+'''
	    from
	        df join expTime on
	        df._time between expTime.start and expTime.end
	    '''
	dfNew = pd.read_sql_query(qry, conn)

	dfNew = dfNew.set_index('experiment').rename(columns={"_time": "min"})#.drop(columns=["messageID"])

	dfNew['min'] = pd.to_datetime(dfNew["min"], format='mixed')
	# dfNew['_min'] = pd.to_datetime(dfNew["_min"])

	#Try resampling for every seconds
	dfNoIndex = dfNew.reset_index()
	# dfNoIndex.head(10)

	by_time = dfNoIndex.groupby([dfNoIndex['experiment'],dfNoIndex[parameter],pd.Grouper(key="min", freq='1s')])['_measurement'].count().reset_index()
	dfAggTime = by_time.rename(columns={"_measurement": "count"})

	print(dfAggTime)

	# Fill the voids
	minTime = dfAggTime.groupby(['experiment']).agg('min').drop(columns=['count'])
	maxTime = dfAggTime.groupby(['experiment']).agg('max').drop(columns=['count'])
	# maxTime.head(10)

	minMax = minTime.merge(maxTime, on=['experiment']).rename(columns={"min_x": "min", "min_y": "max", parameter+"_x": parameter}).drop(columns=[parameter+'_y']).reset_index()
	# minMax.head(10)

	date_list = pd.date_range(minMax['min'].min(), minMax['max'].max(),freq='1s')

	dates = pd.DataFrame(date_list).rename(columns={0:"_time"})
	dates['count'] = 0

	#Make the db in memory
	conn = sqlite3.connect(':memory:')
	#write the tables
	minMax.to_sql('minMax', conn, index=False)
	dates.to_sql('dates', conn, index=False)

	qry = '''
	    select  
	        dates._time as _time,
	        minMax.'''+parameter+''',
	        minMax.experiment,
	        dates.count
	    from
	        dates join minMax on
	        dates._time between minMax.min and minMax.max
	    '''
	dfFill = pd.read_sql_query(qry, conn)
	# dfFill.head(10)
	# print(dfFill)
	# print(dfAggTime)

	#write the tables
	dfFill.to_sql('fill', conn, index=False)
	dfAggTime.to_sql('df', conn, index=False)

	qry = '''
	    select
	       experiment,
	       '''+parameter+''',
	       _time as min,
	       count
	    from fill
	    where fill._time not in (SELECT DISTINCT min FROM df)
	    '''
	dfMissingTime = pd.read_sql_query(qry, conn).reset_index(drop=True).drop_duplicates()
	# print(dfMissingTime)
	dfMissingTime['min'] = pd.to_datetime(dfMissingTime["min"], format='mixed')
	# dfNew['min'] = pd.to_datetime(dfNew["min"], format='mixed')

	dfAggTime =  pd.concat([dfMissingTime.reset_index(drop=True), dfAggTime.reset_index(drop=True)]).sort_values(by=['min'])
	df = dfAggTime.drop(columns=['min'])

	avgPropExp = df.groupby(['experiment']).agg('mean')
	avgPropExp.reset_index(inplace=True)
	avgPropExp = avgPropExp.drop(columns=['experiment'])

	# avgPropExp.head(10)
	print(avgPropExp)

	avgProp = avgPropExp.groupby([parameter]).agg({'count':['mean','std']})
	avgProp.columns = avgProp.columns.droplevel(0)
	avgProp.reset_index(inplace=True)

	print(avgProp)
	return avgProp

	# avgProp.head(100)

def plotBandwidth(df, topology, parameter, parameter_name):

	y_labels = df[parameter].astype(str).to_numpy()
	# print(y_labels)

	plt.style.use('ggplot')
	# kwargs = dict(color=['hotpink'], alpha=0.9)#, density=True)

	fig, ax = plt.subplots(constrained_layout=False)
	ax.grid(alpha=0.4)

	fig.tight_layout()
	fig.subplots_adjust(left=0.19, bottom=0.09, right=0.98, top=0.92, wspace=0.17, hspace=0.17)

	plt.rcParams.update({'figure.figsize':(7,5), 'figure.dpi':100})

	plt.gca().set(title='Bandwidth by '+parameter_name, ylabel='parameter', xlabel="Time [s]")

	plt.rcParams.update({'font.size': 14})
	# plt.hist([x1, x2, x6], **kwargs, label=['GS 1 topic', 'Vanilla', 'Squelching'])
	# plt.legend(loc='upper center', bbox_to_anchor=(0.58, 1))
	# plt.gca().xaxis.set_major_locator(ticker.MultipleLocator(2))

	plt.xticks(fontsize=18)
	plt.yticks(fontsize=16)

	fig.set_size_inches(11, 4.3)

	#Set y axis labels
	ax.barh(y_labels, df['mean'], xerr=df['std'], align='center')
	ax.set_yticks(y_labels)
	ax.invert_yaxis()  # labels read top-to-bottom
	ax.set_xlabel('Propagation time [μs]')
	ax.set_ylabel(parameter_name)


	fig.savefig('./figures/bandwidth_'+topology+'_'+parameter+'.pdf', format='pdf', facecolor='white', edgecolor='none', bbox_inches='tight', dpi=600)
	fig.savefig('./figures/bandwidth_'+topology+'_'+parameter+'.png', format='png', facecolor='white', edgecolor='none', bbox_inches='tight', dpi=600)


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
start_time = 1692015275
end_time = 1692166070

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
#   Start and end for reference and executions
#############################################
ref = experiments.loc[experiments["parameter"] == "reference"]
start_reference = ref["startUnix"].min().astype(int)
end_reference = ref["endUnix"].max().astype(int)

reference 		= functions.from_influx(url, token, org, "deliverMessage", start_reference, end_reference, "_measurement")
reference_rpc 	= functions.from_influx(url, token, org, "recvRPC", start_reference, end_reference, "_measurement")

#############################################
#	Loop to procces each graph
#############################################
# oldMeasurement = ''
with open('bargraphs_parameters.csv', 'r') as file:
    reader = csv.DictReader(file)
    data = list(reader)
    for graph in data:
        print(graph)
            
        #Get start and end time
        par = experiments.loc[experiments["parameter"] == graph["parameter"]]
        start_query = par["startUnix"].min().astype(int)
        end_query = par["endUnix"].max().astype(int)

        print("Influx query")
        rpc  		= functions.from_influx(url, token, org, "recvRPC", start_query, end_query, '_measurement')
        received 	= functions.from_influx(url, token, org, "deliverMessage",start_query, end_query, '_measurement')

        # print(rpc)
        # print(received)

        exp = experiments.loc[experiments['topology'] == graph['topology']]
        exp = exp.loc[exp['parameter'] == graph['parameter']]
        exp = pd.concat([exp, ref])

        received 	= pd.concat([received, reference])
        rpc 		= pd.concat([rpc, reference_rpc])

        df = calcBandwidth(received, rpc, exp, graph['parameter'])
        print("Data tratead")

        plotBandwidth(df,graph['topology'], graph['parameter'],graph['parameter_name'])

        print("Graph generated")
