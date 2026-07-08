# Problem Maker

Problem maker is a program to generate/simulate various kinds of node problems. It is used in NPD e2e tests to verify NPD's behavior when node problems happen:
1. NPD should report the problems correctly.
2. NPD should survive the problems as much as possible.

**Problem maker is NOT intended to be used in any other places. And please do NOT run this directly on your workstation.** Problem makers can cause real OS failures, and the current ones write fake kernel messages to `/dev/kmsg` as root, which pollutes the kernel log and can trigger any real monitoring watching it.

You shouldn't need to run it anyways. If you want to test NPD, it's best to run NPD e2e test.

## Developing/Testing Problem Maker

If you want to enrich the problems that problem maker can generate, you may want to run it to test the behavior. Then the recommended way for running it is to run it in a VM:
```
sudo problem-maker --help
sudo problem-maker --problem DockerHung
sudo problem-maker --problem Ext4FilesystemError
```

Problem maker tries to generate real node problems, and can cause real node failures. And when we do not have a good way to generate the problems, we instruct problem maker to simulate problems by injecting logs. Generating real problems is preferred over injecting logs when the node can survive them. This is because when kernel is upgraded, log patterns can change. NPD e2e tests is supposed to verify whether NPD can correctly understand the tested kernel.
