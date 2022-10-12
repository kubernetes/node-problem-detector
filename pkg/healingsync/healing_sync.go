/*
@Copyright (C) Ctyun Inc. All rights reserved.
@Date : 2022/9/27 18:17
@Author : linshw
@Descriptions ：
*/

package healingsync

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"

	"k8s.io/node-problem-detector/pkg/custompluginmonitor"
	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
	"k8s.io/node-problem-detector/pkg/problemdetector"
	"k8s.io/node-problem-detector/pkg/problemmetrics"
	"k8s.io/node-problem-detector/pkg/systemlogmonitor"
	watchertypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	systemlogtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util/fileutil"
)

const (
	ScriptPath = "/npd/"
	ConfigPath = "/npd/configs/"
)

type CronService struct {
	monitors    map[string]types.Monitor //如考虑不兼容配置项，可废弃
	taskChn     chan *problemdetector.ProblemSync
	interval    int
	url         string
	curMonitors map[int64]*Healing
}

func NewCronService(monitors map[string]types.Monitor, interval int, url string) *CronService {
	return &CronService{
		monitors:    monitors,
		taskChn:     make(chan *problemdetector.ProblemSync, 128),
		interval:    interval,
		url:         url,
		curMonitors: make(map[int64]*Healing),
	}
}

func (c *CronService) GetChn() <-chan *problemdetector.ProblemSync {
	return c.taskChn
}

func (c *CronService) Run(termCh <-chan error) error {
	if c.interval < 1 || c.url == "" {
		glog.Info("cron service is disable.")
		return nil
	}

	timer := time.NewTicker(time.Duration(c.interval) * time.Second)
	if err := fileutil.CreatDir(ConfigPath); err != nil {
		glog.Errorf("create directory failed path:%s, err:%v", ConfigPath, err)
		panic(err)
	}

	glog.V(2).Infof("cron service stack detail:%v", c)
	for {
		select {
		case <-termCh:
			return nil
		case <-timer.C:
			c.getMonitorConfig()
		}
	}
}

func (c *CronService) getMonitorConfig() {
	glog.V(2).Infof("start get monitor config. url:%s", c.url)
	resp, err := httpRequest(c.url)
	if err != nil {
		if err1 := problemmetrics.GlobalProblemMetricsManager.IncrementSyncCounter("sync config failed", 1); err1 != nil {
			glog.Errorf("Failed to update sync counter metrics for sync failed: %v", err1)
		}
		glog.Errorf("Failed to get monitor config url:%s, err:%v", c.url, err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if err1 := problemmetrics.GlobalProblemMetricsManager.IncrementSyncCounter("read response body error", 1); err1 != nil {
			glog.Errorf("Failed to update sync counter metrics for read response failed: %v", err1)
		}
		glog.Errorf("Read response body failed, err:%s", err.Error())
		return
	}
	glog.V(2).Infof("resp body:%s", string(body))

	var tasks HealingTasks
	if err := json.Unmarshal(body, &tasks); err != nil {
		if err1 := problemmetrics.GlobalProblemMetricsManager.IncrementSyncCounter("unmarshal config failed", 1); err1 != nil {
			glog.Errorf("Failed to update sync counter metrics for unmarshal config failed: %v", err1)
		}
		glog.Errorf("Unmarshal response body failed, err:%s", err.Error())
		return
	}

	extra := make(map[int64]int64)
	for k, _ := range tasks.Items {
		extra[tasks.Items[k].MonitorId] = tasks.Items[k].MonitorId
		if cur, ok := c.curMonitors[tasks.Items[k].MonitorId]; ok {
			if cur != nil && cur.Version == tasks.Items[k].Version {
				continue
			}
		}
		if tasks.Items[k].MonitorType == LogMode {
			_ = c.genLogMonitor(&tasks.Items[k])
		} else if tasks.Items[k].MonitorType == CustomPluginMode {
			_ = c.genCustomPlugin(&tasks.Items[k])
		}
	}

	for k, v := range c.curMonitors {
		if _, ok := extra[k]; !ok && v != nil {
			delTask := &problemdetector.ProblemSync{
				ConfigName: strconv.FormatInt(v.MonitorId, 10),
				IsDelete:   true,
			}
			c.taskChn <- delTask
			delete(c.curMonitors, k)
		}
	}
}

func (c *CronService) genLogMonitor(one *Healing) error {
	task := &problemdetector.ProblemSync{
		ConfigName: strconv.FormatInt(one.MonitorId, 10),
		Version:    one.Version,
		IsDelete:   false,
	}

	config := systemlogmonitor.MonitorConfig{
		WatcherConfig: watchertypes.WatcherConfig{
			Plugin:  "log",
			LogPath: one.LogPath,
		},
		Source: strconv.FormatInt(one.MonitorId, 10),
		Rules:  make([]systemlogtypes.Rule, 0),
	}

	patternByte, err := base64.StdEncoding.DecodeString(one.Pattern)
	if err != nil {
		return err
	}

	rule := systemlogtypes.Rule{
		Type:    types.Type(one.RulesType),
		Reason:  one.RulesReason,
		Pattern: string(patternByte),
	}
	config.Rules = append(config.Rules, rule)

	configByte, err := json.Marshal(&config)
	if err != nil {
		return err
	}

	filename := ConfigPath + strconv.FormatInt(one.MonitorId, 10) + ".json"
	if err := os.WriteFile(filename, configByte, os.ModePerm); err != nil {
		return err
	}

	task.Monitors = systemlogmonitor.NewLogMonitorOrDie(filename)

	c.taskChn <- task
	return nil
}

func (c *CronService) genCustomPlugin(one *Healing) error {
	task := &problemdetector.ProblemSync{
		ConfigName: strconv.FormatInt(one.MonitorId, 10),
		Version:    one.Version,
		IsDelete:   false,
	}

	pluginGlobalConfig := cpmtypes.NewPluginGlobalConfig()
	pluginGlobalConfig.InvokeIntervalString = &one.Interval
	filename := ScriptPath + strconv.FormatInt(one.MonitorId, 10)

	config := cpmtypes.CustomPluginConfig{
		Plugin:             "custom",
		Source:             strconv.FormatInt(one.MonitorId, 10),
		PluginGlobalConfig: pluginGlobalConfig,
	}
	rule := &cpmtypes.CustomRule{
		Type:          types.Type(one.RulesType),
		Reason:        one.RulesReason,
		Args:          one.Args,
		Path:          filename,
		TimeoutString: &one.Timeout,
	}
	config.Rules = append(config.Rules, rule)

	//write script
	scriptByte, err := base64.StdEncoding.DecodeString(one.Pattern)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filename, scriptByte, os.ModePerm); err != nil {
		return err
	}

	//write config
	configByte, err := json.Marshal(&config)
	if err != nil {
		return err
	}
	name := ConfigPath + strconv.FormatInt(one.MonitorId, 10) + ".json"
	if err := os.WriteFile(name, configByte, os.ModePerm); err != nil {
		return err
	}

	task.Monitors = custompluginmonitor.NewCustomPluginMonitorOrDie(name)

	c.taskChn <- task
	return nil
}
