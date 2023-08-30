import matplotlib.pyplot as plt
import numpy as np
import os
import re
import pandas as pd
import seaborn as sns

width = 6.5
height = 4

def getStorePath():
    # 获取当前Python文件的绝对路径
    current_file_path = os.path.abspath(__file__)
    # 获取当前Python文件所在的目录
    current_directory = os.path.dirname(current_file_path)
    # # 定义图片的保存路径
    # image_path = os.path.join(current_directory, 'figs')
    return current_directory

def motivation_security():
    '''
    用于在motivation中说明分片安全性不足的图
    '''
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (9, 4)

    ## 第一幅图，随着交易注入，失效分片导致的累积失效交易数量逐渐增多
    plt.subplot(1,2,1)
    shardNums = np.array([1,2,4,8])
    data = np.array([
        [500,1000,1500,2000,2500,3000,3500,4000,4500,5000],
        [369,743,1108,1456,1813,2262,2664,3017,3387,3736],
        [207,469,719,918,1118,1512,1797,2017,2236,2463],
        [118,297,452,563,683,1047,1270,1400,1537,1685]
        ])
    for i in range(len(shardNums)):
        shardData = data[i]
        plt.plot(data[0], data[i], label="ShardNum="+str(shardNums[i]))
    plt.xticks(fontsize=16)
    plt.yticks(fontsize=16)
    plt.xlabel('# of all txs', fontsize=18)
    plt.ylabel('# of directly affected txs', fontsize=18)
    plt.legend(fontsize=15, loc='center', bbox_to_anchor=(0.32,0.75))
    plt.tight_layout()
    # plt.savefig('experiments/figs/motivation_security_1.png')
    # plt.show()

    ## 第二幅图，随着分片数量增加，攻击者令单个分片失效所需的力量逐渐减少
    plt.subplot(1,2,2)
    ### 假设分片内运行PBFT协议
    shardNums = np.array(range(1,9))
    powers_to_hinder = 1/3 / shardNums
    powers_to_control = 2/3 / shardNums
    plt.plot(shardNums, powers_to_hinder, label="block_consensus")
    plt.plot(shardNums, powers_to_control, label="control_consensus")
    plt.xticks(fontsize=16)
    plt.yticks(fontsize=16)
    plt.xlabel('# of shards', fontsize=18)
    plt.ylabel('power needed to attack one shard    ', fontsize=14)
    plt.legend(fontsize=15, loc='center', bbox_to_anchor=(0.58,0.75))
    plt.tight_layout()

    plt.savefig('experiments/figs/motivation_security.png')
    plt.show()


# 数据来源：experiments/results/tps
def TPS(dataPath):
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    plt.grid('on', alpha=0.5, axis='y')
    shards = [2,8,16,24,32]
    # tps = [236,551,1236,1491,2365,2899]
    dataPath = os.path.join(getStorePath(), dataPath)
    tps,_,_ = getData(shards,dataPath)
    
    # colors = ['#26697d', '#82c6d5', '#bceed4', '#f2e2a9', '#fffff3']
    bar = plt.bar(x=shards, height=tps, width=3, alpha=0.8)
    plt.bar_label(bar, labels=tps, fontsize=16, padding=10)
    plt.plot(shards, tps, '.-', color='tab:red', markersize=12)
    
    plt.xticks(shards, fontsize=16)
    plt.yticks(fontsize=16)
    # plt.yticks([0,2000,4000,6000,8000,10000],['0','2k','4k','6k','8k','10k'],fontsize=16)
    # plt.yticks([0,1000,2000,3000,4000,5000], ['0', '1k', '2k', '3k', '4k', '5k'], fontsize=16)
    plt.xlabel('Number of shards', fontsize=18)
    plt.ylabel('Throughput (TX/s)', fontsize=18)
    ax = plt.gca()
    ax.spines['right'].set_visible(False)
    ax.spines['top'].set_visible(False)
    ax.spines['left'].set_visible(False)
    plt.tick_params(bottom=False, top=False, left=False, right=False)
    plt.tight_layout()
    # plt.savefig('figs/throughput.pdf')
    storePath = os.path.join(getStorePath(), 'figs/throughput.png')
    plt.savefig(storePath)
    plt.show()

def getLatDetail(shards, dataPath):
    path = os.path.join(getStorePath(), dataPath)
    lats = []
    for s in shards:
        filename = path + 'shard' + str(s) + '.csv'
        with open(filename) as f:
            content = f.read()
        lat = content.split(',')
        lat = list(map(int,lat))
        lats = lats + [lat]
    # print(len(lats))
    # print(lats[0][:10])
    return lats

# 数据来源：experiments/results/tps
def Latency(dataPath):
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    # plt.style.use('_mpl-gallery')
    shards = [2,8,16,24,32]
    
    lat_detail = getLatDetail(shards, dataPath)
    palette = ['#26697d', '#82c6d5', '#bceed4', '#f2e2a9', '#fffff3']
    # palette1 = ['#512324', '#d9de5e', '#888201', '#46a8a8', '#a8c7cb']
    x = []
    y = []
    for i in range(len(lat_detail)):
        x = x + [shards[i]] * len(lat_detail[i])
        y = y + lat_detail[i]
    print(len(x),len(y))
    sns.boxplot(x=x,y=y, width=0.5, palette=palette)

    dataPath = os.path.join(getStorePath(), dataPath)
    _,latency,_ = getData(shards,dataPath)
    plt.plot([0,1,2,3,4], latency, '.-', color='tab:red', markersize=12, label='Average Latency')
    plt.xticks(fontsize=16)
    plt.yticks(fontsize=16)
    plt.ylim(0,130)
    plt.xlabel('Number of shards', fontsize=18)
    plt.ylabel('Latency (s)', fontsize=18)
    plt.legend(fontsize=16, loc='center', bbox_to_anchor=(0.57,0.9))
    plt.tight_layout()
    # plt.savefig('figs/latency.pdf')
    storePath = os.path.join(getStorePath(), 'figs/latency.png')
    plt.savefig(storePath)
    plt.show()


def getData(shards, path):
    tpss = []
    lats = []
    rollbackRates = []
    for s in shards:
        filename = path + 'shard' + str(s) + '.log'
        with open(filename) as f:
            content = f.read()
        pat = re.compile(r'GetThroughtPutAndLatency.*caller')
        text = pat.findall(content)[0]
        pat = re.compile(r'thrput=([0-9]*\.?[0-9]+) avlatency=([0-9]*\.?[0-9]+) rollbackRate=([0-9]*\.?[0-9]+) overloads="\[(.*)\]"')
        res = pat.findall(text)

        # print(text)
        # print(res)
        tps = int(float(res[0][0]))
        latency = float(res[0][1])
        rollbackRate = float(res[0][2])
        # workload = res[0][3].split()
        # print(tps,latency,rollbackRate,workload)
        lats = lats + [latency]
        tpss = tpss + [tps]
        rollbackRates = rollbackRates + [rollbackRate]
    # print(lats)
    return tpss,lats,rollbackRates

# 数据来源：experiments/results/workload
def Workload4shard():
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    labels = ['S = 8', 'S = 16', 'S = 32']
    colors = ['tab:red', 'tab:orange', 'tab:blue']
    shards = [
        [i for i in range(8)],
        [i for i in range(16)],
        [i for i in range(32)]
        ]
    workload = [
        [66000,65290,55413,54240,62878,60163,64411,64720],
        [28000,26372,24578,25042,26838,20752,27012,27251,28000,27902,21906,20246,25209,28000,26167,27645],
        [11000,7508,9089,8633,8110,7493,10309,10570,11000,9596,8756,7494,8651,9544,7490,11000,9064,11000,9087,10579,10717,8287,10646,10375,10341,11000,8063,8017,10656,11000,9929,10151]
        ]
    for i in range(len(shards)):
        plt.plot(shards[i], workload[i], label=labels[i], color=colors[i])

    
    plt.xticks([],fontsize=16)
    # plt.yticks([0,5e3,1e4,1.5e4,2e4,2.5e4,3e4],[0,'5k','10k','15k','20k','25k','30k'],fontsize=16)
    plt.yticks([0,1e4,2e4,3e4,4e4,5e4,6e4],[0,'10k','20k','30k','40k','50k','60k'],fontsize=16)
    plt.xlabel('\nSequantial IDs of shards', fontsize=18)
    plt.ylabel('Workload (# of TX)', fontsize=18)
    plt.legend(fontsize=16,loc='center', bbox_to_anchor=(0.7,0.7))
    plt.tight_layout()
    # plt.savefig('figs/workload.pdf')
    storePath = os.path.join(getStorePath(), 'figs/workload.png')
    plt.savefig(storePath)
    plt.show()

# 数据来源：experiments/results/
def RollBackRate(dataPath):
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    plt.grid('on', alpha=0.5)
    shards = [8,16,32]
    rollback_height = [6,8,12]
    labels = ['S = 8', 'S = 16', 'S = 32']
    # colors = ['#82c6d5', '#f2e2a9', '#fffff3']
    colors = ['tab:brown', 'tab:red', 'tab:green']
    linestyles = ['x-', '.-', 's-']
    markersizes = [7,12,6]
    
    for i in range(len(shards)):
        rates = []
        for j in range(len(rollback_height)):
            path = os.path.join(getStorePath(), dataPath+'height'+str(rollback_height[j])+'/')
            cur_shard = [shards[i]]
            tps,latency,rate = getData(cur_shard, path)
            rates.append(rate)
        plt.plot(rollback_height, rates, linestyles[i], label=labels[i], color=colors[i], markersize=markersizes[i])
    plt.xticks(rollback_height, fontsize=16)
    plt.yticks(fontsize=16)
    plt.xlabel('Height to rollback', fontsize=18)
    plt.ylabel('Prop. of TXs rollback', fontsize=18)
    plt.legend(fontsize=16, loc='center',bbox_to_anchor=(0.7,0.45))
    plt.tight_layout()
    # plt.savefig('figs/rollbackRate.pdf')
    storePath = os.path.join(getStorePath(), 'figs/rollbackRate.png')
    plt.savefig(storePath)
    plt.show()


def Reconfig(dataPath):  
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    
    shards = [8]
    reconfig_height = [2,4,6,8]
    labels = ['S = 8']
    colors = ['tab:blue', 'tab:red', 'tab:green', 'tab:brown']
    linestyles = ['.-', 's-', '*-', 'x-']
    markersizes = [8,6,8,8]

    fig, ax1 = plt.subplots()  # 创建原始的figure和axis
    ax2 = ax1.twinx()  # 创建第二个y轴
    
    for i in range(len(shards)):
        tpss = []
        delays = []
        for j in range(len(reconfig_height)):
            path = os.path.join(getStorePath(), dataPath+'height'+str(reconfig_height[j])+'/')
            cur_shards = [shards[i]]
            tps, delay, _ = getData(cur_shards, path)
            tpss.append(tps)
            delays.append(delay)
            
        ax1.plot(reconfig_height, tpss, linestyles[1], label=labels[i]+" Throughput", color=colors[i], markersize=markersizes[1]) 
        ax2.plot(reconfig_height, delays, linestyles[2], label=labels[i] + " Latency", color=colors[i], markersize=markersizes[2]) 
    
    ax1.set_xticks(reconfig_height)
    ax1.tick_params(axis='both', labelsize=16)
    ax1.set_xlabel('Height to reconfig', fontsize=18)
    ax1.set_ylabel('Throughput (TX/s)', fontsize=18)
    
    ax2.set_ylabel('Delay', fontsize=18)
    ax2.tick_params(axis='y', labelsize=16)

    # 合并两个轴的图例
    lines, labels = ax1.get_legend_handles_labels()
    lines2, labels2 = ax2.get_legend_handles_labels()
    ax2.legend(lines + lines2, labels + labels2, fontsize=14, loc='center',bbox_to_anchor=(0.6,0.45))
    
    plt.tight_layout()
    
    storePath = os.path.join(getStorePath(), 'figs/reconfig.png')
    plt.savefig(storePath)
    plt.show()

def ConfirmHeight(dataPath):
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    
    shards = [8,16,32]
    ConfirmHeight = [0,2,4,6]
    labels = ['S = 8', 'S = 16', 'S = 32']
    colors = ['tab:blue', 'tab:red', 'tab:green']
    linestyles = ['.-', 's-', '*-', 'x-']
    markersizes = [8,6,8,8]

    fig, ax1 = plt.subplots()  # 创建原始的figure和axis
    # ax2 = ax1.twinx()  # 创建第二个y轴
    
    for i in range(len(shards)):
        tpss = []
        delays = []
        for j in range(len(ConfirmHeight)):
            path = os.path.join(getStorePath(), dataPath+'height'+str(ConfirmHeight[j])+'/')
            cur_shards = [shards[i]]
            tps, delay, _ = getData(cur_shards, path)
            tpss.append(tps)
            delays.append(delay)
            
        ax1.plot(ConfirmHeight, tpss, linestyles[i], label=labels[i], color=colors[i], markersize=markersizes[i])
        # ax2.plot(ConfirmHeight, delays, linestyles[2], label=labels[i] + " Latency", color=colors[i], markersize=markersizes[2])
    
    ax1.set_xticks(ConfirmHeight)
    ax1.tick_params(axis='both', labelsize=16)
    ax1.set_xlabel('Height to confirm', fontsize=18)
    ax1.set_ylabel('Throughput (TX/s)', fontsize=18)
    

    ax1.legend(fontsize=14, loc='center',bbox_to_anchor=(0.5,0.55))
    
    plt.tight_layout()
    
    storePath = os.path.join(getStorePath(), 'figs/confirmHeight.png')
    plt.savefig(storePath)
    plt.show()


def main():
    # motivation_security()
    # TPS('results/workload/')
    # Latency('results/workload/')
    # Workload4shard()
    RollBackRate('results/rollbackV2/')
    # Reconfig('results/reconfig/')
    # ConfirmHeight('results/confirmHeight/')

if __name__ == "__main__":
    main()