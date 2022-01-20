/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package local

// follow text is the default configuration template for syncthing local
const LocalSyncConfigXML = `<configuration version="32">
{{ range .Folders }}
<folder id="nh-{{ .Name }}" label="{{ .Name }}" path="{{ .LocalPath }}" type="{{ $.Type }}" 
rescanIntervalS="{{ $.RescanInterval }}" fsWatcherEnabled="true" 
fsWatcherDelayS="1" ignorePerms="false" autoNormalize="true">
	<filesystemType>basic</filesystemType>
	<device id="SJTYMUE-DI3REKX-JCLCRXU-F6UJHCG-XQGHAZJ-5O5D3JR-LALGSBC-TJ4I4QO" introducedBy=""></device>
	<device id="{{$.RemoteDeviceID}}" introducedBy=""></device>
	<minDiskFree unit="%">1</minDiskFree>
	<versioning></versioning>
	<copiers>0</copiers>
	<pullerMaxPendingKiB>0</pullerMaxPendingKiB>
	<hashers>0</hashers>
	<order>random</order>
	<ignoreDelete>{{ $.IgnoreDelete }}</ignoreDelete>
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
<device id="SJTYMUE-DI3REKX-JCLCRXU-F6UJHCG-XQGHAZJ-5O5D3JR-LALGSBC-TJ4I4QO" 
name="local" compression="local" introducer="false" 
skipIntroductionRemovals="false" introducedBy="">
	<address>dynamic</address>
	<paused>false</paused>
	<autoAcceptFolders>false</autoAcceptFolders>
	<maxSendKbps>0</maxSendKbps>
	<maxRecvKbps>0</maxRecvKbps>
	<maxRequestKiB>0</maxRequestKiB>
</device>
<device id="{{.RemoteDeviceID}}" name="remote" compression="metadata" 
introducer="false" skipIntroductionRemovals="false" introducedBy="">
	<address>tcp://{{.RemoteAddress}}</address>
	<paused>false</paused>
	<autoAcceptFolders>false</autoAcceptFolders>
	<maxSendKbps>0</maxSendKbps>
	<maxRecvKbps>0</maxRecvKbps>
	<maxRequestKiB>0</maxRequestKiB>
</device>
<gui enabled="true" tls="false" debugging="false">
	<address>{{.GUIAddress}}</address>
	<apikey>{{.APIKey}}</apikey>
	<user>nocalhost</user>
	<password>{{.GUIPasswordHash}}</password>
	<theme>default</theme>
</gui>
<ldap></ldap>
<options>
	<listenAddress>tcp://{{.ListenAddress}}</listenAddress>
	<globalAnnounceServer>default</globalAnnounceServer>
	<globalAnnounceEnabled>false</globalAnnounceEnabled>
	<localAnnounceEnabled>false</localAnnounceEnabled>
	<maxSendKbps>0</maxSendKbps>
	<maxRecvKbps>0</maxRecvKbps>
	<reconnectionIntervalS>30</reconnectionIntervalS>
	<relaysEnabled>false</relaysEnabled>
	<relayReconnectIntervalM>10</relayReconnectIntervalM>
	<startBrowser>false</startBrowser>
	<natEnabled>true</natEnabled>
	<natLeaseMinutes>60</natLeaseMinutes>
	<natRenewalMinutes>30</natRenewalMinutes>
	<natTimeoutSeconds>10</natTimeoutSeconds>
	<urAccepted>-1</urAccepted>
	<urSeen>3</urSeen>
	<urURL></urURL>
	<urPostInsecurely>false</urPostInsecurely>
	<urInitialDelayS>1800</urInitialDelayS>
	<restartOnWakeup>true</restartOnWakeup>
	<autoUpgradeIntervalH>0</autoUpgradeIntervalH>
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
	<defaultFolderPath></defaultFolderPath>
	<setLowPriority>false</setLowPriority>
	<minHomeDiskFreePct>0</minHomeDiskFreePct>
	<crashReportingEnabled>false</crashReportingEnabled>
</options>
</configuration>`

// follow text is the default configuration template for syncthing remote
const RemoteSyncConfigXML = `<configuration version="32">
{{ range .Folders }}
<folder id="nh-{{ .Name }}" label="{{ .Name }}" path="{{ .RemotePath }}" 
type="sendreceive" rescanIntervalS="{{ $.RescanInterval }}" fsWatcherEnabled="true" 
fsWatcherDelayS="1" ignorePerms="false" autoNormalize="true">
	<filesystemType>basic</filesystemType>
	<device id="SJTYMUE-DI3REKX-JCLCRXU-F6UJHCG-XQGHAZJ-5O5D3JR-LALGSBC-TJ4I4QO" introducedBy=""></device>
	<device id="MDPJNTF-OSPJC65-LZNCQGD-3AWRUW6-BYJULSS-GOCA2TU-5DWWBNC-TKM4VQ5" introducedBy=""></device>
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
<device id="SJTYMUE-DI3REKX-JCLCRXU-F6UJHCG-XQGHAZJ-5O5D3JR-LALGSBC-TJ4I4QO" name="local" 
compression="metadata" introducer="false" skipIntroductionRemovals="false" introducedBy="">
	<address>dynamic</address>
	<paused>false</paused>
	<autoAcceptFolders>false</autoAcceptFolders>
	<maxSendKbps>0</maxSendKbps>
	<maxRecvKbps>0</maxRecvKbps>
	<maxRequestKiB>0</maxRequestKiB>
</device>
<device id="MDPJNTF-OSPJC65-LZNCQGD-3AWRUW6-BYJULSS-GOCA2TU-5DWWBNC-TKM4VQ5" name="remote" 
compression="metadata" introducer="false" skipIntroductionRemovals="false" introducedBy="">
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

// IgnoredFileTemplate ignore file pattern template
// first block is ignored pattern
// that's because we should make sure the ignored pattern with highest priority
// if there is same configuration for synced pattern and ignored pattern,
// the file will be ignored
// such as:
// build
// !build
// then the build file or dir will be ignored
//
// the last block is **
// means, if we did not specify any pattern, no file will be synced
const IgnoredFileTemplate = `
// Whether to enable the parse the gitignore from the home directory 
// While enabled, the configuration of "ignoredPattern" or "syncedPattern" will not take effect.
{{.enableParseFromGitIgnore}}

// Ignored pattern block, the priority of ignored pattern is highest, default is ""
{{.ignoredPattern}}

// Synced pattern block, default is "!**"
{{.syncedPattern}}

// ignored all for basic, and it's lowest priority
**
`
