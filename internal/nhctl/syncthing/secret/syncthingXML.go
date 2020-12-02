/*
Copyright 2020 The Nocalhost Authors.
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

// this is syncthing config file template

package secret

import (
	"bytes"
	"html/template"
	"nocalhost/internal/nhctl/syncthing"
)

const configXML = `<configuration version="32">
{{ range .Folders }}
<folder id="nh-{{ .Name }}" label="{{ .Name }}" path="{{ .RemotePath }}" type="sendreceive" rescanIntervalS="{{ $.RescanInterval }}" fsWatcherEnabled="true" fsWatcherDelayS="1" ignorePerms="false" autoNormalize="true">
    <filesystemType>basic</filesystemType>
    <device id="ABKAVQF-RUO4CYO-FSC2VIP-VRX4QDA-TQQRN2J-MRDXJUC-FXNWP6N-S6ZSAAR" introducedBy=""></device>
    <device id="ATOPHFJ-VPVLDFY-QVZDCF2-OQQ7IOW-OG4DIXF-OA7RWU3-ZYA4S22-SI4XVAU" introducedBy=""></device>
    <minDiskFree unit="%">1</minDiskFree>
    <versioning></versioning>
    <copiers>0</copiers>
    <pullerMaxPendingKiB>0</pullerMaxPendingKiB>
    <hashers>0</hashers>
    <order>random</order>
    <ignoreDelete>false</ignoreDelete>
    <scanProgressIntervalS>2</scanProgressIntervalS>
    <pullerPauseS>0</pullerPauseS>
    <maxConflicts>0</maxConflicts>
    <disableSparseFiles>false</disableSparseFiles>
    <disableTempIndexes>false</disableTempIndexes>
    <paused>false</paused>
    <weakHashThresholdPct>25</weakHashThresholdPct>
    <markerName>.</markerName>
    <useLargeBlocks>false</useLargeBlocks>
</folder>
{{ end }}
<device id="ABKAVQF-RUO4CYO-FSC2VIP-VRX4QDA-TQQRN2J-MRDXJUC-FXNWP6N-S6ZSAAR" name="local" compression="metadata" introducer="false" skipIntroductionRemovals="false" introducedBy="">
    <address>dynamic</address>
    <paused>false</paused>
    <autoAcceptFolders>false</autoAcceptFolders>
    <maxSendKbps>0</maxSendKbps>
    <maxRecvKbps>0</maxRecvKbps>
    <maxRequestKiB>0</maxRequestKiB>
</device>
<device id="ATOPHFJ-VPVLDFY-QVZDCF2-OQQ7IOW-OG4DIXF-OA7RWU3-ZYA4S22-SI4XVAU" name="remote" compression="metadata" introducer="false" skipIntroductionRemovals="false" introducedBy="">
    <address>dynamic</address>
    <paused>false</paused>
    <autoAcceptFolders>false</autoAcceptFolders>
    <maxSendKbps>0</maxSendKbps>
    <maxRecvKbps>0</maxRecvKbps>
    <maxRequestKiB>0</maxRequestKiB>
</device>
<gui enabled="true" tls="false" debugging="false">
    <address>{{ .RemoteGUIAddress }}</address>
    <apikey>{{.APIKey}}</apikey>
    <user>nocalhost</user>
    <password>{{.GUIPasswordHash}}</password>
    <theme>default</theme>
</gui>
<ldap></ldap>
<options>
    <listenAddress>tcp://{{.RemoteAddress}}</listenAddress>
    <globalAnnounceServer>default</globalAnnounceServer>
    <globalAnnounceEnabled>false</globalAnnounceEnabled>
    <localAnnounceEnabled>false</localAnnounceEnabled>
    <localAnnouncePort>21027</localAnnouncePort>
    <localAnnounceMCAddr>[ff12::8384]:21027</localAnnounceMCAddr>
    <maxSendKbps>0</maxSendKbps>
    <maxRecvKbps>0</maxRecvKbps>
    <reconnectionIntervalS>60</reconnectionIntervalS>
    <relaysEnabled>false</relaysEnabled>
    <relayReconnectIntervalM>10</relayReconnectIntervalM>
    <startBrowser>false</startBrowser>
    <natEnabled>false</natEnabled>
    <natLeaseMinutes>60</natLeaseMinutes>
    <natRenewalMinutes>30</natRenewalMinutes>
    <natTimeoutSeconds>10</natTimeoutSeconds>
    <urAccepted>-1</urAccepted>
    <urSeen>3</urSeen>
    <urUniqueID>PDhuWgmF</urUniqueID>
    <urURL></urURL>
    <urPostInsecurely>false</urPostInsecurely>
    <urInitialDelayS>1800</urInitialDelayS>
    <restartOnWakeup>true</restartOnWakeup>
    <autoUpgradeIntervalH>12</autoUpgradeIntervalH>
    <upgradeToPreReleases>false</upgradeToPreReleases>
    <keepTemporariesH>24</keepTemporariesH>
    <cacheIgnoredFiles>false</cacheIgnoredFiles>
    <progressUpdateIntervalS>2</progressUpdateIntervalS>
    <limitBandwidthInLan>false</limitBandwidthInLan>
    <minHomeDiskFree unit="%">1</minHomeDiskFree>
    <releasesURL></releasesURL>
    <overwriteRemoteDeviceNamesOnConnect>false</overwriteRemoteDeviceNamesOnConnect>
    <tempIndexMinBlocks>10</tempIndexMinBlocks>
    <trafficClass>0</trafficClass>
    <defaultFolderPath>~</defaultFolderPath>
    <setLowPriority>false</setLowPriority>
    <minHomeDiskFreePct>0</minHomeDiskFreePct>
    <crashReportingEnabled>false</crashReportingEnabled>
</options>
</configuration>`

func GetConfigXML(s *syncthing.Syncthing) ([]byte, error) {
	configTemplate := template.Must(template.New("syncthingConfig").Parse(configXML))
	buf := new(bytes.Buffer)
	if err := configTemplate.Execute(buf, s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
