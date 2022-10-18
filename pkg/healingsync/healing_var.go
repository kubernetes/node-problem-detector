/*
@Copyright (C) Ctyun Inc. All rights reserved.
@Date : 2022/9/29 15:13
@Author : linshw
@Descriptions ：
*/

package healingsync

type MonitorType int

const (
	LogMode MonitorType = iota + 1
	CustomPluginMode
)

type HealingTasks struct {
	RequestId string        `json:"requestId"`
	Status    HealingStatus `json:"status"`
	Works     []Healing     `json:"works"`
}

type Healing struct {
	MonitorId   int64       `json:"monitorId"`   //监控对象唯一id
	MonitorType MonitorType `json:"monitorType"` //1是日志，2是脚本
	Interval    string      `json:"interval"`    //执行间隔,如30s，2m
	LogPath     string      `json:"logPath"`
	Pattern     string      `json:"pattern"` //正则表达式，base64编码
	RulesReason string      `json:"rulesReason"`
	RulesType   string      `json:"rulesType"`
	Version     string      `json:"version"`
	Script      string      `json:"script"` //脚本内容，base64编码
	Args        []string    `json:"args"`
	Timeout     string      `json:"timeout"`
}

type HealingStatus struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
