/*
Copyright 2018 Aladdin Blockchain Technologies Ltd
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

package svc

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const configTxSpecTemplateText = `{{- $domain := .Domain}}---
################################################################################
#
#   Section: Organizations
#
#   - This section defines the different organizational identities which will
#   be referenced later in the configuration.
#
################################################################################
Organizations:

    # SampleOrg defines an MSP using the sampleconfig.  It should never be used
    # in production but may be used as a template for other definitions
    - &OrdererOrg
      # DefaultOrg defines the organization which is used in the sampleconfig
      # of the fabric.git development environment
      Name: OrdererMSP

      # ID to load the MSP definition as
      ID: OrdererMSP

      # MSPDir is the filesystem path which contains the MSP configuration
      MSPDir: crypto-config/ordererOrganizations/{{.Domain}}/msp

{{range $index, $org := .OrganizationSpecs}}
    - &{{$org.Name}}
        # DefaultOrg defines the organization which is used in the sampleconfig
        # of the fabric.git development environment
        Name: {{$org.Name}}MSP

        # ID to load the MSP definition as
        ID: {{$org.ID}}

        # MSPDir is the filesystem path which contains the MSP configuration
        MSPDir: crypto-config/peerOrganizations/{{$org.Name | ToLower}}.{{$domain}}/msp

        AnchorPeers:
           # AnchorPeers defines the location of peers which can be used
           # for cross org gossip communication.  Note, this value is only
           # encoded in the genesis block in the Application section context
           - Host: {{$org.Anchor}}.{{ $org.Name | ToLower }}.{{$domain}}
             Port: 7051

{{end}}

################################################################################
#
#   SECTION: Capabilities
#
#   - This section defines the capabilities of fabric network. This is a new
#   concept as of v1.1.0 and should not be utilized in mixed networks with
#   v1.0.x peers and orderers.  Capabilities define features which must be
#   present in a fabric binary for that binary to safely participate in the
#   fabric network.  For instance, if a new MSP type is added, newer binaries
#   might recognize and validate the signatures from this type, while older
#   binaries without this support would be unable to validate those
#   transactions.  This could lead to different versions of the fabric binaries
#   having different world states.  Instead, defining a capability for a channel
#   informs those binaries without this capability that they must cease
#   processing transactions until they have been upgraded.  For v1.0.x if any
#   capabilities are defined (including a map with all capabilities turned off)
#   then the v1.0.x peer will deliberately crash.
#
################################################################################
Capabilities:
    # Channel capabilities apply to both the orderers and the peers and must be
    # supported by both.  Set the value of the capability to true to require it.
    Global: &ChannelCapabilities
        # V1.1 for Global is a catchall flag for behavior which has been
        # determined to be desired for all orderers and peers running v1.0.x,
        # but the modification of which would cause incompatibilities.  Users
        # should leave this flag set to true.
        V1_1: true

    # Orderer capabilities apply only to the orderers, and may be safely
    # manipulated without concern for upgrading peers.  Set the value of the
    # capability to true to require it.
    Orderer: &OrdererCapabilities
        # V1.1 for Order is a catchall flag for behavior which has been
        # determined to be desired for all orderers running v1.0.x, but the
        # modification of which  would cause incompatibilities.  Users should
        # leave this flag set to true.
        V1_1: true

    # Application capabilities apply only to the peer network, and may be safely
    # manipulated without concern for upgrading orderers.  Set the value of the
    # capability to true to require it.
    Application: &ApplicationCapabilities
        # V1.1 for Application is a catchall flag for behavior which has been
        # determined to be desired for all peers running v1.0.x, but the
        # modification of which would cause incompatibilities.  Users should
        # leave this flag set to true.
        V1_1: true

################################################################################
#
#   SECTION: Application
#
#   - This section defines the values to encode into a config transaction or
#   genesis block for application related parameters
#
################################################################################
Application: &ApplicationDefaults

    # Organizations is the list of orgs which are defined as participants on
    # the application side of the network
    Organizations:

################################################################################
#
#   SECTION: Orderer
#
#   - This section defines the values to encode into a config transaction or
#   genesis block for orderer related parameters
#
################################################################################
Orderer: &OrdererDefaults

    # Orderer Type: The orderer implementation to start
    # Available types are "solo" and "kafka"
    OrdererType: solo

    Addresses:
        - orderer.{{.Domain}}:7050

    # Batch Timeout: The amount of time to wait before creating a batch
    BatchTimeout: 2s

    # Batch Size: Controls the number of messages batched into a block
    BatchSize:

        # Max Message Count: The maximum number of messages to permit in a batch
        MaxMessageCount: 10

        # Absolute Max Bytes: The absolute maximum number of bytes allowed for
        # the serialized messages in a batch.
        AbsoluteMaxBytes: 99 MB

        # Preferred Max Bytes: The preferred maximum number of bytes allowed for
        # the serialized messages in a batch. A message larger than the preferred
        # max bytes will result in a batch larger than preferred max bytes.
        PreferredMaxBytes: 512 KB

    Kafka:
        # Brokers: A list of Kafka brokers to which the orderer connects
        # NOTE: Use IP:port notation
        Brokers:
            - 127.0.0.1:9092

    # Organizations is the list of orgs which are defined as participants on
    # the orderer side of the network
    Organizations:

################################################################################
#
#   Profile
#
#   - Different configuration profiles may be encoded here to be specified
#   as parameters to the configtxgen tool
#
################################################################################
Profiles:

    OrdererGenesis:
        Capabilities:
            <<: *ChannelCapabilities
        Orderer:
            <<: *OrdererDefaults
            Organizations:
                - *OrdererOrg
            Capabilities:
                <<: *OrdererCapabilities
        Consortiums:
        {{- $allOrgs := .OrganizationSpecs}}
        {{- range $index, $consortium := .ConsortiumSpecs}}
            {{$consortium.Name}}:
                Organizations:
                {{- range $index, $org := $allOrgs}} 
                  - *{{$org.Name}}
                {{- end}}
        {{- end}}

{{- range $index, $consortium := .ConsortiumSpecs}}  
    
    {{- range $index, $channelSpec := $consortium.ChannelSpecs}}

    {{$channelSpec.Name}}:
        Consortium: {{$consortium.Name}}
        Application:
            <<: *ApplicationDefaults
            Organizations:
			{{- range $index, $org := $channelSpec.Organizations}}
                - *{{$org}}
            {{- end}} 
            Capabilities:
                <<: *ApplicationCapabilities
    
    {{- end}}
{{- end}}
`

func generateConfigTxSpec(spec NetworkSpec) error {

	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
	}

	tpl := template.Must(template.New("ConfigTxSpec").Funcs(funcMap).Parse(configTxSpecTemplateText))
	configtxYml := filepath.Join(spec.NetworkPath, "configtx.yaml")
	f, err := os.Create(configtxYml)
	if err != nil {
		return err
	}
	err = tpl.Execute(f, spec)
	if err != nil {
		return err
	}

	return nil
}

const configTxExecSriptText = `#!/bin/bash

# Generate genesis block
configtxgen -profile OrdererGenesis -outputBlock ./channel-artefacts/genesis.block

{{- range $index1, $consortium := .ConsortiumSpecs}}
  {{range $index2, $channelSpec := $consortium.ChannelSpecs}}
# {{$channelSpec.Name}} configuration
configtxgen -profile {{$channelSpec.Name}} -outputCreateChannelTx ./channel-artefacts/{{$channelSpec.Name | ToLower }}.tx -channelID {{$channelSpec.Name | ToLower}}
      {{range $index3, $org := $channelSpec.Organizations}}
# Anchor peer for {{$channelSpec.Name}} for transaction of {{$org}}
configtxgen -profile {{$channelSpec.Name}} -outputAnchorPeersUpdate ./channel-artefacts/{{$org}}MSPanchors_{{$channelSpec.Name | ToLower }}.tx -channelID {{$channelSpec.Name | ToLower}} -asOrg {{$org}}MSP
      {{end}}
  {{- end}}
{{end}}
`

func generateConfigTxExecScript(spec NetworkSpec) error {

	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
	}

	tpl := template.Must(template.New("ConfigTxExec").Funcs(funcMap).Parse(configTxExecSriptText))
	generateConfigtxAssetSh := filepath.Join(spec.NetworkPath, "generateConfigTx.sh")
	f, err := os.Create(generateConfigtxAssetSh)
	if err != nil {
		return err
	}

	err = os.Chmod(generateConfigtxAssetSh, 0777)
	if err != nil {
		return err
	}

	err = tpl.Execute(f, spec)
	if err != nil {
		return err
	}
	return nil
}

func execConfigTxExecScript(spec NetworkSpec) error {
	cmd := []string{"./generateConfigTx.sh"}
	if err := RunCryptoConfigContainer(spec.NetworkPath, "configtx", "hyperledger/fabric-tools", cmd); err != nil {
		return err
	}
	return nil
}

// CreateChannelArtefacts produces channel artefacts
func CreateChannelArtefacts(spec NetworkSpec) error {

	if err := generateConfigTxExecScript(spec); err != nil {
		return err
	}
	if err := generateConfigTxSpec(spec); err != nil {
		return err
	}
	if err := execConfigTxExecScript(spec); err != nil {
		return err
	}

	return nil
}
