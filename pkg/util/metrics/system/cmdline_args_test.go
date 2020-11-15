package system

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmdlineStats(t *testing.T) {
	testcases := []struct {
		name                  string
		fakeCmdlineFilePath   string
		expectedCmdlineArgs   []CmdlineArg
		unExpectedCmdlineArgs []CmdlineArg
	}{
		{
			name:                "default_cos",
			fakeCmdlineFilePath: "testdata/cmdline_args_key_cos.txt",
			expectedCmdlineArgs: []CmdlineArg{
				{
					Key:   "console",
					Value: "ttyS0",
				},
				{
					Key:   "boot",
					Value: "local",
				},
				{
					Key: "cros_efi",
				},
			},
			unExpectedCmdlineArgs: []CmdlineArg{
				{
					Key:   "hashstart",
					Value: "4077568",
				},
				{
					Key: "vroot",
				},
			},
		},
		{
			name:                "sample",
			fakeCmdlineFilePath: "testdata/cmdline_args_sample.txt",
			expectedCmdlineArgs: []CmdlineArg{
				{
					Key:   "key1",
					Value: "value1",
				},
				{
					Key:   "key3",
					Value: "value2 value3",
				},
				{
					Key: "key2",
				},
			},
			unExpectedCmdlineArgs: []CmdlineArg{
				{
					Key:   "value2",
					Value: "value3",
				},
				{
					Key: "value3",
				},
			},
		},
	}
	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			originalCmdlineFilePath := cmdlineFilePath
			defer func() {
				cmdlineFilePath = originalCmdlineFilePath
			}()

			cmdlineFilePath = test.fakeCmdlineFilePath
			cmdlineArgs, err := CmdlineArgs()
			if err != nil {
				t.Errorf("Unexpected error retrieving cmdlineArgs: %v\nCmdlineArgsFilePath: %s\n", err, cmdlineFilePath)
			}
			for _, expectedCmdlineArg := range test.expectedCmdlineArgs {
				assert.Contains(t, cmdlineArgs, expectedCmdlineArg, "Failed to find cmdlineArgs: %v\n", expectedCmdlineArg)
			}
			for _, unExpectedCmdlineArg := range test.unExpectedCmdlineArgs {
				assert.NotContains(t, cmdlineArgs, unExpectedCmdlineArg, "Unpected expected cmdlinearg found: %v\n", unExpectedCmdlineArg)
			}
		})
	}
}

func TestCmdlineStats_String(t *testing.T) {
	v := CmdlineArg{
		Key:   "test",
		Value: "test",
	}
	e := `{"key":"test","value":"test"}`
	assert.Equal(t,
		e, fmt.Sprintf("%v", v), "CmdlineArg string is invalid: %v", v)

}
