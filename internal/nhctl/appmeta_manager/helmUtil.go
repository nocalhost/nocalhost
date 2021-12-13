/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package appmeta_manager

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	"nocalhost/internal/nhctl/appmeta"
	profile2 "nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"strings"
)

// Release describes a deployment of a chart, together with the chart
// and the variables used to deploy that chart.
type Release struct {
	// Name is the name of the release
	Name string `json:"name,omitempty"`
	// Info provides information about a release
	Info *Info `json:"info,omitempty"`
	// Config is the set of extra Values added to the chart.
	// These values override the default values inside of the chart.
	Config map[string]interface{} `json:"config,omitempty"`
	// Manifest is the string representation of the rendered template.
	Manifest string `json:"manifest,omitempty"`
	// Version is an int which represents the revision of the release.
	Version int `json:"version,omitempty"`
	// Namespace is the kubernetes namespace of the release.
	Namespace string `json:"namespace,omitempty"`
	// Labels of the release.
	// Disabled encoding into Json cause labels are stored in storage driver metadata field.
	Labels map[string]string `json:"-"`
}

// Info describes release information.
type Info struct {
	// FirstDeployed is when the release was first deployed.
	FirstDeployed string `json:"first_deployed,omitempty"`
	// LastDeployed is when the release was last deployed.
	LastDeployed string `json:"last_deployed,omitempty"`
	// Deleted tracks when this object was deleted.
	Deleted string `json:"deleted"`
	// Description is human-friendly "log entry" about this release.
	Description string `json:"description,omitempty"`
	// Status is the current state of the release
	Status Status `json:"status,omitempty"`
	// Contains the rendered templates/NOTES.txt if available
	Notes string `json:"notes,omitempty"`
}

// Status is the status of a release
type Status string

var b64 = base64.StdEncoding

var magicGzip = []byte{0x1f, 0x8b, 0x08}

func GetRlsNameFromKey(key string) (string, error) {
	nsAndKeyWithoutPrefix := strings.Split(key, "sh.helm.release.v1.")

	if len(nsAndKeyWithoutPrefix) == 0 {
		return "", errors.New("Invalid Helm Key while delete event watched, not contain 'sh.helm.release.v1.'. ")
	}

	var keyWithoutPrefix = nsAndKeyWithoutPrefix[len(nsAndKeyWithoutPrefix)-1]
	elems := strings.Split(keyWithoutPrefix, ".v")

	if len(elems) != 2 {
		return "", errors.New("Invalid Helm Key while delete event watched. ")
	}
	return elems[0], nil
}

func tryDelAppFromHelmRelease(appName, ns string, configBytes []byte) error {
	meta := GetApplicationMeta(ns, appName, configBytes)
	if meta.IsNotInstall() || meta.IsInstalling() {
		return nil
	}

	return errors.Wrap(
		meta.DoWithTempOperator(
			configBytes, func() error {
				if err := meta.Delete(); err != nil {
					return err
				} else {
					log.TLogf("Watcher", "Uninstall application %s by managed helm", appName)
					return nil
				}
			},
		),
		"Error while uninstall application for helm release",
	)
}

func tryNewAppFromHelmRelease(releaseStr, ns string, configBytes []byte) error {
	release, err := DecodeRelease(releaseStr)
	if err != nil {
		return err
	}

	if release == nil {
		return errors.New("decode release str but fail")
	}

	// there is a special case that
	// helm uninstall the Application
	// and do not delete the cm or secret
	if release.Info.Deleted != "" {
		return tryDelAppFromHelmRelease(release.Name, ns, configBytes)
	}

	meta := GetApplicationMeta(ns, release.Name, configBytes)
	if meta.IsInstalled() || meta.IsInstalling() {
		return nil
	}

	return errors.Wrap(
		meta.DoWithTempOperator(
			configBytes,
			func() error {

				if err := meta.OneTimesInitial(
					func(meta *appmeta.ApplicationMeta) {
						meta.ApplicationType = appmeta.HelmLocal
						meta.HelmReleaseName = release.Name
						meta.Application = release.Name
						meta.Config = &profile2.NocalHostAppConfigV2{
							Migrated:          true,
							ApplicationConfig: profile2.ApplicationConfig{},
							ConfigProperties:  profile2.ConfigProperties{},
						}
					}, true,
				); err != nil {
					log.TLogf(
						"Watcher", "Initial application '%s' by managed helm fail, Error: %s",
						release.Name, err,
					)
					return err
				}

				return nil
			},
		),
		"Error while new application for helm release",
	)
}

// DecodeRelease decodes the bytes of data into a release
// type. Data must contain a base64 encoded gzipped string of a
// valid release, otherwise an error is returned.
func DecodeRelease(data string) (*Release, error) {
	// base64 decode string
	b, err := b64.DecodeString(data)
	if err != nil {
		return nil, err
	}

	// For backwards compatibility with releases that were stored before
	// compression was introduced we skip decompression if the
	// gzip magic header is not found
	if bytes.Equal(b[0:3], magicGzip) {
		r, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		b2, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		b = b2
	}

	var rls Release
	// unmarshal release object bytes
	if err := json.Unmarshal(b, &rls); err != nil {
		return nil, err
	}
	return &rls, nil
}
