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
    plt.plot(shardNums, powers_to_hinder, label="level1: hinder")
    plt.plot(shardNums, powers_to_control, label="Level2: manipulate")
    plt.xticks(fontsize=16)
    plt.yticks(fontsize=16)
    plt.xlabel('# of shards', fontsize=18)
    plt.ylabel('power needed to attack one shard    ', fontsize=14)
    plt.legend(fontsize=15, loc='center', bbox_to_anchor=(0.58,0.75))
    plt.tight_layout()

    plt.savefig('experiments/figs/motivation_security.png')
    plt.show()


def TPS(dataPath):
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    plt.grid('on', alpha=0.5, axis='y')
    shards = [2,8,16,24,32]
    # tps = [236,551,1236,1491,2365,2899]
    dataPath = os.path.join(getStorePath(), dataPath)
    tps,_,_,_ = getData(shards,dataPath)
    
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
    storePath = os.path.join(getStorePath(), 'figs', 'throughput.png')
    # 确保目录存在
    if not os.path.exists(os.path.dirname(storePath)):
        os.makedirs(os.path.dirname(storePath))
    plt.savefig(storePath)
    plt.show()

def getLatDetail(group, dataPath):
    path = os.path.join(getStorePath(), dataPath)
    
    files = []
    if "tps" in path:
        for num in group:
            files.append(path + 'shard' + str(num) + 'run1.csv')
    elif "injectSpeed" in path:
        for num in group:
            files.append(path + 'shard32inject' + str(num) + '.csv')
    
    lats = []
    for filename in files:
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
    _,latency,_,_ = getData(shards,dataPath)
    plt.plot([0,1,2,3,4], latency, '.-', color='tab:red', markersize=12, label='Average Latency')
    plt.xticks(fontsize=16)
    plt.yticks(fontsize=16)
    # plt.ylim(0,130)
    plt.xlabel('Number of shards', fontsize=18)
    plt.ylabel('Latency (s)', fontsize=18)
    plt.legend(fontsize=16, loc='center', bbox_to_anchor=(0.57,0.9))
    plt.tight_layout()
    # plt.savefig('figs/latency.pdf')
    storePath = os.path.join(getStorePath(), 'figs', 'latency.png')
    # 确保目录存在
    if not os.path.exists(os.path.dirname(storePath)):
        os.makedirs(os.path.dirname(storePath))
    plt.savefig(storePath)
    plt.show()

def Workload4shard(dataPath):
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    labels = ['S = 8', 'S = 16', 'S = 32']
    colors = ['tab:red', 'tab:orange', 'tab:blue']
    shards = [8,16,32]
    shardIndexs = []
    for shard in shards:
        shardIndexs.append([i for i in range(shard)])
    
    dataPath = os.path.join(getStorePath(), dataPath)
    _,_,_,workloadStrs = getData(shards, dataPath)
    workloads = []
    for workload in workloadStrs:
        workload = [int(num) for num in workload[1:-1].split()]
        workloads.append(workload)
    # print(workloads)
    for i in range(len(shards)):
        plt.plot(shardIndexs[i], workloads[i], label=labels[i], color=colors[i])

    
    plt.xticks([],fontsize=16)
    plt.yticks([0,5e4,10e4,15e4,20e4],[0,'5w','10w','15w','20w'],fontsize=16)
    # plt.yticks([0,3e4,2e4,3e4,4e4,5e4,6e4],[0,'10k','20k','30k','40k','50k','60k'],fontsize=16)
    plt.xlabel('\nSequantial IDs of shards', fontsize=18)
    plt.ylabel('Workload (# of TX)', fontsize=18)
    plt.legend(fontsize=16,loc='center', bbox_to_anchor=(0.7,0.7))
    plt.tight_layout()
    # plt.savefig('figs/workload.pdf')
    storePath = os.path.join(getStorePath(), 'figs', 'workload.png')
    # 确保目录存在
    if not os.path.exists(os.path.dirname(storePath)):
        os.makedirs(os.path.dirname(storePath))
    plt.savefig(storePath)
    plt.show()


def getData(group, path):
    tpss = []
    lats = []
    rollbackRates = []
    workloads = []
    
    files = []
    if "tps" in path or "rollback" in path:
        for num in group:
            files.append(path + 'shard' + str(num) + 'run1.log')
    elif "injectSpeed" in path:
        for num in group:
            files.append(path + 'shard32inject' + str(num) + '.log')
        
    for filename in files:
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
        workload = res[0][3]
        # print(tps,latency,rollbackRate,workload)
        lats = lats + [latency]
        tpss = tpss + [tps]
        rollbackRates = rollbackRates + [rollbackRate]
        workloads = workloads + [workload]
    # print(lats)
    return tpss,lats,rollbackRates,workloads

def TPS1(dataPath):
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    plt.grid('on', alpha=0.5, axis='y')
    speeds = [2000,4000,8000,10000,12000,16000]
    dataPath = os.path.join(getStorePath(), dataPath)
    tps,_,_,_ = getData(speeds,dataPath)
    
    # colors = ['#26697d', '#82c6d5', '#bceed4', '#f2e2a9', '#fffff3']
    bar = plt.bar(x=speeds, height=tps, width=1000, alpha=0.8)
    plt.bar_label(bar, labels=tps, fontsize=16, padding=10)
    plt.plot(speeds, tps, '.-', color='tab:red', markersize=12)
    
    plt.xticks(speeds, ['2k', '4k', '8k', '10k', '12k', '16k'], fontsize=16)
    plt.yticks(fontsize=16)
    # plt.yticks([0,2000,4000,6000,8000,10000],['0','2k','4k','6k','8k','10k'],fontsize=16)
    # plt.yticks([0,1000,2000,3000,4000,5000], ['0', '1k', '2k', '3k', '4k', '5k'], fontsize=16)
    plt.xlabel('Inject speed (TX/s)', fontsize=18)
    plt.ylabel('Throughput (TX/s)', fontsize=18)
    ax = plt.gca()
    ax.spines['right'].set_visible(False)
    ax.spines['top'].set_visible(False)
    ax.spines['left'].set_visible(False)
    plt.tick_params(bottom=False, top=False, left=False, right=False)
    plt.tight_layout()
    # plt.savefig('figs/throughput.pdf')
    storePath = os.path.join(getStorePath(), 'figs', 'injectSpeed.png')
    # 确保目录存在
    if not os.path.exists(os.path.dirname(storePath)):
        os.makedirs(os.path.dirname(storePath))
    plt.savefig(storePath)
    plt.show()
    
    
def Latency1(dataPath):
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    # plt.style.use('_mpl-gallery')
    speeds = [2000,4000,8000,10000,12000,16000]
    
    palette = ['#26697d', '#82c6d5', '#bceed4', '#f2e2a9', '#fffff3', '#ce7ed4']
    # palette1 = ['#512324', '#d9de5e', '#888201', '#46a8a8', '#a8c7cb']
    
    lat_detail = getLatDetail(speeds, dataPath)
    x = []
    y = []
    for i in range(len(lat_detail)):
        x = x + [speeds[i]] * len(lat_detail[i])
        y = y + lat_detail[i]
    print(len(x),len(y))
    sns.boxplot(x=x,y=y, width=0.5, palette=palette)

    dataPath = os.path.join(getStorePath(), dataPath)
    _,latency,_,_ = getData(speeds,dataPath)
    plt.plot([0,1,2,3,4,5], latency, '.-', color='tab:red', markersize=12, label='Average Latency')
    plt.xticks(fontsize=16)
    plt.yticks(fontsize=16)
    # plt.ylim(0,130)
    plt.xlabel('Inject speed (TX/s)', fontsize=18)
    plt.ylabel('Latency (s)', fontsize=18)
    plt.legend(fontsize=16, loc='center', bbox_to_anchor=(0.27,0.9))
    plt.tight_layout()
    # plt.savefig('figs/latency.pdf')
    storePath = os.path.join(getStorePath(), 'figs', 'injectSpeed_latency.png')
    # 确保目录存在
    if not os.path.exists(os.path.dirname(storePath)):
        os.makedirs(os.path.dirname(storePath))
    plt.savefig(storePath)
    plt.show()

# 数据来源：experiments/results/
def RollBackRate(dataPath):
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    plt.grid('on', alpha=0.5)
    shards = [8,16,32]
    rollback_height = [6,9,12,18]
    labels = ['S = 8', 'S = 16', 'S = 32', 'S = 64']
    # colors = ['#82c6d5', '#f2e2a9', '#fffff3']
    colors = ['tab:orange', 'tab:red', 'tab:green', 'tab:brown']
    linestyles = ['>-', '.-', 's-']
    markersizes = [7,12,6]
    
    for i in range(len(shards)):
        rates = []
        for j in range(len(rollback_height)):
            path = os.path.join(getStorePath(), dataPath+'height'+str(rollback_height[j])+'/')
            cur_shard = [shards[i]]
            tps,latency,rate,_ = getData(cur_shard, path)
            rates.append(rate)
            # rates.append(tps)
        plt.plot(rollback_height, rates, linestyles[i], label=labels[i], color=colors[i], markersize=markersizes[i])
    plt.xticks(rollback_height, fontsize=16)
    plt.yticks(fontsize=16)
    plt.xlabel('Height to rollback', fontsize=18)
    plt.ylabel('Prop. of TXs rollback', fontsize=18)
    # plt.ylabel('Latency (s)', fontsize=18)
    # plt.ylabel('Throughput (TX/s)', fontsize=18)
    plt.legend(fontsize=16, loc='center',bbox_to_anchor=(0.7,0.45))
    plt.tight_layout()
    # plt.savefig('figs/rollbackRate.pdf')
    storePath = os.path.join(getStorePath(), 'figs/rollbackRate.png')
    # storePath = os.path.join(getStorePath(), 'figs/rollbackRate_tps.png')
    plt.savefig(storePath)
    plt.show()
    
# 数据来源：experiments/results/
def RollBackRateV2(dataPath):
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.rcParams['figure.figsize'] = (width, height)
    plt.grid('on', alpha=0.5)
    shards = [8,16,32]
    rollback_height = [6,9,12]
    labels = ['H = 6', 'H = 9', 'H = 12']
    # colors = ['#82c6d5', '#f2e2a9', '#fffff3']
    colors = ['tab:blue', 'tab:purple', 'tab:brown']
    linestyles = ['x-', '.-', 's-']
    markersizes = [7,12,6]
    
    for i in range(len(rollback_height)):
        path = os.path.join(getStorePath(), dataPath+'height'+str(rollback_height[i])+'/')
        _,_,rates,_ = getData(shards, path)
        plt.plot(shards, rates, linestyles[i], label=labels[i], color=colors[i], markersize=markersizes[i])
    plt.xticks(shards, fontsize=16)
    plt.yticks(fontsize=16)
    plt.xlabel('# of shards', fontsize=18)
    plt.ylabel('Prop. of TXs rollback', fontsize=18)
    plt.legend(fontsize=16, loc='center',bbox_to_anchor=(0.7,0.45))
    plt.tight_layout()
    # plt.savefig('figs/rollbackRate.pdf')
    storePath = os.path.join(getStorePath(), 'figs/rollbackRateV2.png')
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
            tps, delay, _, _ = getData(cur_shards, path)
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
            tps, delay, _, _ = getData(cur_shards, path)
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
    # TPS('results/tps/')
    # Latency('results/tps/')
    # TPS1('results/injectSpeed/')
    # Latency1('results/injectSpeed/')
    # Workload4shard('results/workload/')
    RollBackRate('results/rollback/')
    RollBackRateV2('results/rollback/')
    # Reconfig('results/reconfig/')
    # ConfirmHeight('results/confirmHeight/')

if __name__ == "__main__":
    main()