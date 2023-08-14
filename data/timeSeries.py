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

def insert_missing_time(dfAggTime, parameter):
    # Fill the voids
    minTime = dfAggTime.groupby(['experiment']).agg('min').drop(columns=['count'])
    maxTime = dfAggTime.groupby(['experiment']).agg('max').drop(columns=['count'])
    # maxTime.head(10)

    minMax = minTime.merge(maxTime, on=['experiment']).rename(columns={"min_x": "min", "min_y": "max", parameter+"_x": parameter}).reset_index()
    # minMax.head(10)

    date_list = pd.date_range(minMax['min'].min(), minMax['max'].max(),freq='10s')

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
    print(dfAggTime)

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

    df =  pd.concat([dfMissingTime.reset_index(drop=True), dfAggTime.reset_index(drop=True)]).sort_values(by=['min'])
    # dfAggTime.head(200)

    return df

def group_time(df, expRaw, parameter,grouping_key, start, end):

    expTime = expRaw.loc[expRaw['startUnix']>= int(start)].loc[expRaw['endUnix'] <= int(end)]
    print("expTime")
    print(expTime)

    #Make the db in memory
    conn = sqlite3.connect(':memory:')
    #write the tables
    df.to_sql('df', conn, index=False)
    expTime.to_sql('expTime', conn, index=False)

    qry = '''
        select  
            df._time,
            df.'''+grouping_key+''',
            expTime.experiment,
            expTime.'''+parameter+'''
        from
            df join expTime on
            df._time between expTime.start and expTime.end
        '''
    dfNew = pd.read_sql_query(qry, conn)

    print("SQL query -> join")
    print(dfNew)

    dfNew = dfNew.set_index('experiment').rename(columns={"_time": "min"})#.drop(columns=["messageID"])

    dfNew['min'] = pd.to_datetime(dfNew["min"], format='mixed')
    # dfNew['_min'] = pd.to_datetime(dfNew["_min"])

    #Try resampling for every 5 seconds
    dfNoIndex = dfNew.reset_index()
    # dfNoIndex.head(10)

    by_time = dfNoIndex.groupby([dfNoIndex['experiment'],dfNoIndex[parameter],pd.Grouper(key="min", freq='10s')])[grouping_key].count().reset_index()
    dfAggTime = by_time.rename(columns={grouping_key: "count"})

    dfAggTime = insert_missing_time(dfAggTime, parameter)

    #Min datetime of each experiment
    minTime = dfAggTime.groupby(['experiment']).agg('min').drop(columns=[parameter, 'count'])
    print("Min time per experiment")
    print(minTime)

    # minTime.head(20)

    #Join to calculate delta
    dfWithMin = dfAggTime.merge(minTime, on='experiment', how='left').rename(columns={'min_x': '_time', 'min_y': '_min'})

    dfWithMin['_time'] = pd.to_datetime(dfWithMin["_time"], format='mixed')
    dfWithMin['_min'] = pd.to_datetime(dfWithMin["_min"],  format='mixed')

    # #Calculate delta in seconds 
    dfWithMin["delta"] = ((dfWithMin["_time"] - dfWithMin["_min"]) / pd.Timedelta(seconds=1)).astype(int)

    # print(dfWithMin)

    # dfWithMin.head(100)

    #Aggregate by time
    gb = dfWithMin.groupby(['delta','experiment', parameter])['count'].agg(["sum"]).sort_values(by=["experiment", "delta"])
    # gb.columns = gb.columns.droplevel(0)
    # gb.reset_index(level=0, inplace=True)

    print("Grouped by time")
    print(gb)

    # gb.head(100)

    #Average by interval
    intv = gb.groupby([parameter,'delta']).agg(["mean"]).sort_values(by=[parameter, "delta"])
    intv.columns = intv.columns.droplevel(0)#.droplevel(1)
    # intv.reset_index(level=0, inplace=True)
    intv.reset_index(inplace=True)

    print("AVG to plot")
    print(intv)

    return intv

def generate_graph(measurement, measurement_name, topology, intv, parameter, parameter_name):

    plt.style.use('ggplot')
    # kwargs = dict(color=['hotpink'], alpha=0.9)#, density=True)

    fig, ax = plt.subplots(constrained_layout=False)
    ax.grid(alpha=0.4)

    fig.tight_layout()
    fig.subplots_adjust(left=0.19, bottom=0.09, right=0.98, top=0.92, wspace=0.17, hspace=0.17)

    plt.rcParams.update({'figure.figsize':(7,5), 'figure.dpi':100})

    plt.gca().set(title=measurement_name+' per '+parameter_name+' '+topology, ylabel=measurement, xlabel="Time [s]")

    plt.rcParams.update({'font.size': 14})
    # plt.hist([x1, x2, x6], **kwargs, label=['GS 1 topic', 'Vanilla', 'Squelching'])
    # plt.legend(loc='upper center', bbox_to_anchor=(0.58, 1))
    # plt.gca().xaxis.set_major_locator(ticker.MultipleLocator(2))

    plt.xticks(fontsize=18)
    plt.yticks(fontsize=16) 

    fig.set_size_inches(11, 4.3)

    metrics = intv[parameter].drop_duplicates().to_numpy()
    print(metrics)

    for metric in np.nditer(metrics):
        aux = intv.loc[intv[parameter] == metric]
        interv = aux[parameter].drop_duplicates().item()
        plt.plot(aux["delta"], aux["mean"], label=interv)

    ax.legend(loc="upper left", bbox_to_anchor=[0, 1],
                     ncols=2, shadow=True, title="Legend", fancybox=True)

    fig.savefig('./figures/'+measurement+'_'+topology+'_'+parameter+'.pdf', format='pdf', facecolor='white', edgecolor='none', bbox_inches='tight', dpi=600)
    fig.savefig('./figures/'+measurement+'_'+topology+'_'+parameter+'.png', format='png', facecolor='white', edgecolor='none', bbox_inches='tight', dpi=600)

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
start_time = 1691678444
end_time = 1691767949

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

# measurement = "duplicateMessage"
# measurement_name = "Message Overhead"
# topology = "unl"
# parameter = "interval"
# parameter_name = "interval"
# grouping_key = "messageID"

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
    # print(data)
    for graph in data:
        print(graph)
        # if graph['measurement'] != oldMeasurement:
        print("Influx query")
        traces = functions.from_influx(url, token, org, graph['measurement'], graph['start'], graph['end'], graph['grouping_key'])

        exp = experiments.loc[experiments['topology'] == graph['topology']]
        gb = group_time(traces, exp, graph['parameter'], graph['grouping_key'], graph['start'], graph['end'])

        print("Data tratead")

        generate_graph(graph['measurement'], graph['measurement_name'], graph['topology'], gb, graph['parameter'],graph['parameter_name'])

        print("Graph generated")

        # oldMeasurement = graph['measurement']


# #get topology
# exp = experiments.loc[experiments['topology'] == topology]

# gb = group_time(traces, exp, parameter, grouping_key)

# generate_graph(measurement, measurement_name, topology, gb, parameter,parameter_name)