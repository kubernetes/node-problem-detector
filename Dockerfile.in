# Copyright 2016 The Kubernetes Authors All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM @BASEIMAGE@
MAINTAINER Random Liu <lantaol@google.com>

RUN clean-install bash

# Avoid symlink of /etc/localtime.
RUN test -h /etc/localtime && rm -f /etc/localtime && cp /usr/share/zoneinfo/UTC /etc/localtime || true

ADD ./bin/node-problem-detector /node-problem-detector
ADD config /config
ENTRYPOINT ["/node-problem-detector", "--system-log-monitors=/config/kernel-monitor.json"]
