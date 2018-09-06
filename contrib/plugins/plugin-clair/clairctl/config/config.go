/*

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.

CODE FROM https://github.com/jgsqware/clairctl

*/
package config

import (
	"fmt"
	"net"
	"strings"

	"github.com/jgsqware/xnet"
	"github.com/spf13/viper"
)

var IsLocal = false
var Insecure = false
var NoClean = false

var ImageName string

type reportConfig struct {
	Path, Format string
}
type clairConfig struct {
	URI              string
	Port, HealthPort int
	Report           reportConfig
}
type authConfig struct {
	InsecureSkipVerify bool
}
type clairctlConfig struct {
	IP, Interface, TempFolder string
	Port                      int
}
type docker struct {
	InsecureRegistries []string
}

type config struct {
	Clair    clairConfig
	Auth     authConfig
	Clairctl clairctlConfig
	Docker   docker
}

func TmpLocal() string {
	return viper.GetString("clairctl.tempFolder")
}

type Login struct {
	Username string
	Password string
}

//LocalServerIP return the local clairctl server IP
func LocalServerIP() (string, error) {
	localPort := viper.GetString("clairctl.port")
	localIP := viper.GetString("clairctl.ip")
	localInterfaceConfig := viper.GetString("clairctl.interface")

	if localIP == "" {
		fmt.Printf("retrieving interface for local IP\n")
		var err error
		var localInterface net.Interface
		localInterface, err = translateInterface(localInterfaceConfig)
		if err != nil {
			return "", fmt.Errorf("retrieving interface: %v", err)
		}

		localIP, err = xnet.IPv4(localInterface)
		if err != nil {
			return "", fmt.Errorf("retrieving interface ip: %v", err)
		}
	}
	return strings.TrimSpace(localIP) + ":" + localPort, nil
}

func translateInterface(localInterface string) (net.Interface, error) {
	if localInterface != "" {
		fmt.Printf("interface provided, looking for %s\n", localInterface)
		netInterface, err := net.InterfaceByName(localInterface)
		if err != nil {
			return net.Interface{}, err
		}
		return *netInterface, nil
	}

	fmt.Printf("no interface provided, looking for docker0\n")
	netInterface, err := net.InterfaceByName("docker0")
	if err != nil {
		fmt.Printf("docker0 not found, looking for first connected broadcast interface\n")
		interfaces, err := net.Interfaces()
		if err != nil {
			return net.Interface{}, err
		}

		i, err := xnet.First(xnet.Filter(interfaces, xnet.IsBroadcast), xnet.HasAddr)
		if err != nil {
			return net.Interface{}, err
		}
		return i, nil
	}

	return *netInterface, nil

}
