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

# import functions


# Function to parse configurations from go config file
def _parse_line(line):
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
    client = InfluxDBClient(url=url, token=token, org=org,  timeout=30_000)

    # write_api = client.write_api(write_options=SYNCHRONOUS)
    query_api = client.query_api()

    data_frame = query_api.query_data_frame('from(bucket: "gs") '
                                        ' |> range(start: '+str(start_time)+', stop:'+str(end_time)+') '
                                        ' |> filter(fn: (r) => r._measurement == "'+measurement+'") '
                                        ' |> group(columns: ["_measurement", "_field"], mode: "by") '
                                        ' |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")')
    client.close()

    # df = data_frame.drop(columns=['result', 'table','_start', '_stop', '_measurement', 'topic', 'receivedFrom']).sort_values(by=["_time"]).reset_index(drop=True)
    df = data_frame[['_time', grouping_key]].sort_values(by=["_time"]).reset_index(drop=True)
    df["_time"] = pd.to_datetime(df["_time"])

    return df

def group_time(df, expTime, parameter,grouping_key):
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

    dfNew = dfNew.set_index('experiment').rename(columns={"_time": "min"})#.drop(columns=["messageID"])

    dfNew['min'] = pd.to_datetime(dfNew["min"], format='mixed')
    # dfNew['_min'] = pd.to_datetime(dfNew["_min"])

    #Try resampling for every 5 seconds
    dfNoIndex = dfNew.reset_index()
    # dfNoIndex.head(10)

    by_time = dfNoIndex.groupby([dfNoIndex['experiment'],dfNoIndex[parameter],pd.Grouper(key="min", freq='10s')])[grouping_key].count().reset_index()
    dfAggTime = by_time.rename(columns={grouping_key: "count"})

    #Min datetime of each experiment
    minTime = dfAggTime.groupby(['experiment']).agg('min').drop(columns=[parameter, 'count'])

    # minTime.head(20)

    #Join to calculate delta
    dfWithMin = dfAggTime.merge(minTime, on='experiment', how='left').rename(columns={'min_x': '_time', 'min_y': '_min'})

    dfWithMin['_time'] = pd.to_datetime(dfWithMin["_time"], format='mixed')
    dfWithMin['_min'] = pd.to_datetime(dfWithMin["_min"],  format='mixed')

    # #Calculate delta in seconds 
    dfWithMin["delta"] = ((dfWithMin["_time"] - dfWithMin["_min"]) / pd.Timedelta(seconds=1)).astype(int)

    # dfWithMin.head(100)

    #Aggregate by time
    gb = dfWithMin.groupby(['delta','experiment', parameter])['count'].agg(["sum"]).sort_values(by=["experiment", "delta"])
    # gb.columns = gb.columns.droplevel(0)
    # gb.reset_index(level=0, inplace=True)

    # gb.head(100)

    #Average by interval
    intv = gb.groupby([parameter,'delta']).agg(["mean"]).sort_values(by=["interval", "delta"])
    intv.columns = intv.columns.droplevel(0)#.droplevel(1)
    # intv.reset_index(level=0, inplace=True)
    intv.reset_index(inplace=True)

    return intv

def generate_graph(measurement, measurement_name, topology, intv, parameter, parameter_name):
    metrics = intv[parameter].drop_duplicates().to_numpy()

    plt.style.use('ggplot')
    # kwargs = dict(color=['hotpink'], alpha=0.9)#, density=True)

    fig, ax = plt.subplots(constrained_layout=False)
    ax.grid(alpha=0.4)

    fig.tight_layout()
    fig.subplots_adjust(left=0.19, bottom=0.09, right=0.98, top=0.92, wspace=0.17, hspace=0.17)

    plt.rcParams.update({'figure.figsize':(7,5), 'figure.dpi':100})

    plt.gca().set(title=parameter_name+' per '+measurement_name+' '+topology, ylabel=measurement, xlabel="Time [s]")

    plt.rcParams.update({'font.size': 14})
    # plt.hist([x1, x2, x6], **kwargs, label=['GS 1 topic', 'Vanilla', 'Squelching'])
    # plt.legend(loc='upper center', bbox_to_anchor=(0.58, 1))
    # plt.gca().xaxis.set_major_locator(ticker.MultipleLocator(2))

    plt.xticks(fontsize=18)
    plt.yticks(fontsize=16) 

    fig.set_size_inches(11, 4.3)

    for metric in np.nditer(metrics):
        aux = intv.loc[intv[parameter] == metric]
        interv = aux[parameter].drop_duplicates().item()
        plt.plot(aux["delta"], aux["mean"], label=interv)

    ax.legend(loc="upper left", bbox_to_anchor=[0, 1],
                     ncols=2, shadow=True, title="Legend", fancybox=True)

    fig.savefig('./figures/'+measurement+'_'+topology+'_'+parameter+'.pdf', format='pdf', facecolor='white', edgecolor='none', bbox_inches='tight', dpi=600)
    fig.savefig('./figures/'+measurement+'_'+topology+'_'+parameter+'.png', format='png', facecolor='white', edgecolor='none', bbox_inches='tight', dpi=600)

    # plt.show()

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
start_time = 1690980845
end_time = 1690985064

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

measurement = "duplicateMessage"
measurement_name = "Message Overhead"
topology = "unl"
parameter = "interval"
parameter_name = "interval"
grouping_key = "messageID"

#############################################
#	Query experiments file and influx to get all the data
#############################################
experiments = experiment(start_time, end_time, './experiments.csv')
traces = from_influx(url, token, org, measurement, start_time, end_time, grouping_key)

#############################################
#	Loop to procces each graph
#############################################

#get topology
exp = experiments.loc[experiments['topology'] == topology]

gb = group_time(traces, exp, parameter, grouping_key)

generate_graph(measurement, measurement_name, topology, gb, parameter,parameter_name)