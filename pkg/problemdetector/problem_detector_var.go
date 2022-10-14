/*
@Copyright (C) Ctyun Inc. All rights reserved.
@Date : 2022/9/29 10:00
@Author : linshw
@Descriptions ：
*/

package problemdetector

import "k8s.io/node-problem-detector/pkg/types"

type ProblemSync struct {
	Monitors   types.Monitor
	ConfigName string
	Version    string
	IsDelete   bool
}
