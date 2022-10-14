/*
@Copyright (C) Ctyun Inc. All rights reserved.
@Date : 2022/9/30 15:56
@Author : linshw
@Descriptions ：
*/

package fileutil

import "os"

// FileIsExist check file is exist
func FileIsExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

func CreatDir(path string) error {
	if !FileIsExist(path) {
		err := os.MkdirAll(path, os.ModeDir)
		return err
	}
	return nil
}

// DeleteFile 删除文件
func DeleteFile(filename string) error {
	if !FileIsExist(filename) {
		//	文件不存在
		return nil
	}
	return os.Remove(filename)
}
