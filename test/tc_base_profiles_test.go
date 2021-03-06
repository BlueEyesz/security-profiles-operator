/*
Copyright 2020 The Kubernetes Authors.

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

package e2e_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func (e *e2e) testCaseBaseProfile([]string) {
	const baseProfilePath = "examples/baseprofile.yaml"
	const helloProfile = `
apiVersion: security-profiles-operator.x-k8s.io/v1alpha1
kind: SeccompProfile
metadata:
  name: hello
spec:
  defaultAction: SCMP_ACT_ERRNO
  targetWorkload: hello
  baseProfileName: runc-v1.0.0-rc92
  syscalls:
  - action: SCMP_ACT_ALLOW
    names:
    - arch_prctl
    - set_tid_address
    - exit_group
`
	const helloPod = `
apiVersion: v1
kind: Pod
metadata:
  name: hello
spec:
  containers:
  - image: hello-world
    name: hello
    resources: {}
  securityContext:
    seccompProfile:
      type: Localhost
      localhostProfile: operator/%s/hello/hello.json
  restartPolicy: Never
`
	e.kubectl("create", "-f", baseProfilePath)
	defer e.kubectl("delete", "-f", baseProfilePath)

	e.logf("Creating hello profile")
	helloProfileFile, err := ioutil.TempFile(os.TempDir(), "hello-profile*.yaml")
	e.Nil(err)
	defer os.Remove(helloProfileFile.Name())
	_, err = helloProfileFile.Write([]byte(helloProfile))
	e.Nil(err)
	err = helloProfileFile.Close()
	e.Nil(err)
	e.kubectl("create", "-f", helloProfileFile.Name())
	defer e.kubectl("delete", "-f", helloProfileFile.Name())

	e.logf("Creating hello-world pod")
	helloPodFile, err := ioutil.TempFile(os.TempDir(), "hello-pod*.yaml")
	e.Nil(err)
	defer os.Remove(helloPodFile.Name())

	namespace := e.getCurrentContextNamespace(defaultNamespace)
	_, err = helloPodFile.Write([]byte(fmt.Sprintf(helloPod, namespace)))
	e.Nil(err)
	err = helloPodFile.Close()
	e.Nil(err)
	e.kubectl("create", "-f", helloPodFile.Name())
	defer e.kubectl("delete", "pod", "hello")

	e.logf("Waiting for test pod to be initialized")
	e.kubectl("wait", "--for", "condition=initialized", "pod", "hello")

	output := e.kubectl("get", "pod", "hello")
	for strings.Contains(output, "ContainerCreating") {
		output = e.kubectl("get", "pod", "hello")
	}

	e.logf("Testing that container is launched without runtime permission errors")
	output = e.kubectl("describe", "pod", "hello")
	e.NotContains(output, "Error: failed to start containerd task")

	e.logf("Testing that container ran successfully")
	output = e.kubectl("logs", "hello")
	e.Contains(output, "Hello from Docker!")
}
