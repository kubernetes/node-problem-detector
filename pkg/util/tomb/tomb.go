/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tomb

// Tomb is used to control the lifecycle of a goroutine.
type Tomb struct {
	stop chan struct{}
	done chan struct{}
}

// NewTomb creates a new tomb.
func NewTomb() *Tomb {
	return &Tomb{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
}

// Stop is used to stop the goroutine outside.
func (t *Tomb) Stop() {
	close(t.stop)
	<-t.done
}

// Stopping is used by the goroutine to tell whether it should stop.
func (t *Tomb) Stopping() <-chan struct{} {
	return t.stop
}

// Done is used by the goroutine to inform that it has stopped.
func (t *Tomb) Done() {
	close(t.done)
}
