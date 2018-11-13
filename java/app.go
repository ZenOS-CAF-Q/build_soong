// Copyright 2015 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package java

// This file contains the module types for compiling Android apps.

import (
	"strings"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/tradefed"
)

func init() {
	android.RegisterModuleType("android_app", AndroidAppFactory)
	android.RegisterModuleType("android_test", AndroidTestFactory)
}

// AndroidManifest.xml merging
// package splits

type appProperties struct {
	// path to a certificate, or the name of a certificate in the default
	// certificate directory, or blank to use the default product certificate
	Certificate *string

	// paths to extra certificates to sign the apk with
	Additional_certificates []string

	// If set, create package-export.apk, which other packages can
	// use to get PRODUCT-agnostic resource data like IDs and type definitions.
	Export_package_resources *bool

	// Specifies that this app should be installed to the priv-app directory,
	// where the system will grant it additional privileges not available to
	// normal apps.
	Privileged *bool

	// list of resource labels to generate individual resource packages
	Package_splits []string

	// Names of modules to be overridden. Listed modules can only be other binaries
	// (in Make or Soong).
	// This does not completely prevent installation of the overridden binaries, but if both
	// binaries would be installed by default (in PRODUCT_PACKAGES) the other binary will be removed
	// from PRODUCT_PACKAGES.
	Overrides []string
}

type AndroidApp struct {
	Library
	aapt

	certificate certificate

	appProperties appProperties

	extraLinkFlags []string
}

func (a *AndroidApp) ExportedProguardFlagFiles() android.Paths {
	return nil
}

func (a *AndroidApp) ExportedStaticPackages() android.Paths {
	return nil
}

func (a *AndroidApp) ExportedManifest() android.Path {
	return a.manifestPath
}

var _ AndroidLibraryDependency = (*AndroidApp)(nil)

type certificate struct {
	pem, key android.Path
}

func (a *AndroidApp) DepsMutator(ctx android.BottomUpMutatorContext) {
	a.Module.deps(ctx)
	if !Bool(a.properties.No_framework_libs) && !Bool(a.properties.No_standard_libs) {
		a.aapt.deps(ctx, sdkContext(a))
	}
}

func (a *AndroidApp) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	a.generateAndroidBuildActions(ctx)
}

func (a *AndroidApp) generateAndroidBuildActions(ctx android.ModuleContext) {
	linkFlags := append([]string(nil), a.extraLinkFlags...)

	hasProduct := false
	for _, f := range a.aaptProperties.Aaptflags {
		if strings.HasPrefix(f, "--product") {
			hasProduct = true
		}
	}

	// Product characteristics
	if !hasProduct && len(ctx.Config().ProductAAPTCharacteristics()) > 0 {
		linkFlags = append(linkFlags, "--product", ctx.Config().ProductAAPTCharacteristics())
	}

	// Product AAPT config
	for _, aaptConfig := range ctx.Config().ProductAAPTConfig() {
		linkFlags = append(linkFlags, "-c", aaptConfig)
	}

	// Product AAPT preferred config
	if len(ctx.Config().ProductAAPTPreferredConfig()) > 0 {
		linkFlags = append(linkFlags, "--preferred-density", ctx.Config().ProductAAPTPreferredConfig())
	}

	// TODO: LOCAL_PACKAGE_OVERRIDES
	//    $(addprefix --rename-manifest-package , $(PRIVATE_MANIFEST_PACKAGE_NAME)) \

	a.aapt.buildActions(ctx, sdkContext(a), linkFlags...)

	// apps manifests are handled by aapt, don't let Module see them
	a.properties.Manifest = nil

	var staticLibProguardFlagFiles android.Paths
	ctx.VisitDirectDeps(func(m android.Module) {
		if lib, ok := m.(AndroidLibraryDependency); ok && ctx.OtherModuleDependencyTag(m) == staticLibTag {
			staticLibProguardFlagFiles = append(staticLibProguardFlagFiles, lib.ExportedProguardFlagFiles()...)
		}
	})

	staticLibProguardFlagFiles = android.FirstUniquePaths(staticLibProguardFlagFiles)

	a.Module.extraProguardFlagFiles = append(a.Module.extraProguardFlagFiles, staticLibProguardFlagFiles...)
	a.Module.extraProguardFlagFiles = append(a.Module.extraProguardFlagFiles, a.proguardOptionsFile)

	if ctx.ModuleName() != "framework-res" {
		a.Module.compile(ctx, a.aaptSrcJar)
	}

	c := String(a.appProperties.Certificate)
	switch {
	case c == "":
		pem, key := ctx.Config().DefaultAppCertificate(ctx)
		a.certificate = certificate{pem, key}
	case strings.ContainsRune(c, '/'):
		a.certificate = certificate{
			android.PathForSource(ctx, c+".x509.pem"),
			android.PathForSource(ctx, c+".pk8"),
		}
	default:
		defaultDir := ctx.Config().DefaultAppCertificateDir(ctx)
		a.certificate = certificate{
			defaultDir.Join(ctx, c+".x509.pem"),
			defaultDir.Join(ctx, c+".pk8"),
		}
	}

	certificates := []certificate{a.certificate}
	for _, c := range a.appProperties.Additional_certificates {
		certificates = append(certificates, certificate{
			android.PathForSource(ctx, c+".x509.pem"),
			android.PathForSource(ctx, c+".pk8"),
		})
	}

	packageFile := android.PathForModuleOut(ctx, "package.apk")

	CreateAppPackage(ctx, packageFile, a.exportPackage, a.outputFile, certificates)

	a.outputFile = packageFile

	if ctx.ModuleName() == "framework-res" {
		// framework-res.apk is installed as system/framework/framework-res.apk
		ctx.InstallFile(android.PathForModuleInstall(ctx, "framework"), ctx.ModuleName()+".apk", a.outputFile)
	} else if Bool(a.appProperties.Privileged) {
		ctx.InstallFile(android.PathForModuleInstall(ctx, "priv-app"), ctx.ModuleName()+".apk", a.outputFile)
	} else {
		ctx.InstallFile(android.PathForModuleInstall(ctx, "app"), ctx.ModuleName()+".apk", a.outputFile)
	}
}

func AndroidAppFactory() android.Module {
	module := &AndroidApp{}

	module.Module.deviceProperties.Optimize.Enabled = proptools.BoolPtr(true)
	module.Module.deviceProperties.Optimize.Shrink = proptools.BoolPtr(true)

	module.Module.properties.Instrument = true
	module.Module.properties.Installable = proptools.BoolPtr(true)

	module.AddProperties(
		&module.Module.properties,
		&module.Module.deviceProperties,
		&module.Module.protoProperties,
		&module.aaptProperties,
		&module.appProperties)

	module.Prefer32(func(ctx android.BaseModuleContext, base *android.ModuleBase, class android.OsClass) bool {
		return class == android.Device && ctx.Config().DevicePrefer32BitApps()
	})

	InitJavaModule(module, android.DeviceSupported)
	return module
}

type appTestProperties struct {
	Instrumentation_for *string
}

type AndroidTest struct {
	AndroidApp

	appTestProperties appTestProperties

	testProperties testProperties

	testConfig android.Path
	data       android.Paths
}

func (a *AndroidTest) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if String(a.appTestProperties.Instrumentation_for) != "" {
		a.AndroidApp.extraLinkFlags = append(a.AndroidApp.extraLinkFlags,
			"--rename-instrumentation-target-package",
			String(a.appTestProperties.Instrumentation_for))
	}

	a.generateAndroidBuildActions(ctx)

	a.testConfig = tradefed.AutoGenInstrumentationTestConfig(ctx, a.testProperties.Test_config, a.testProperties.Test_config_template, a.manifestPath)
	a.data = ctx.ExpandSources(a.testProperties.Data, nil)
}

func (a *AndroidTest) DepsMutator(ctx android.BottomUpMutatorContext) {
	android.ExtractSourceDeps(ctx, a.testProperties.Test_config)
	android.ExtractSourceDeps(ctx, a.testProperties.Test_config_template)
	android.ExtractSourcesDeps(ctx, a.testProperties.Data)
	a.AndroidApp.DepsMutator(ctx)
}

func AndroidTestFactory() android.Module {
	module := &AndroidTest{}

	module.Module.deviceProperties.Optimize.Enabled = proptools.BoolPtr(true)

	module.Module.properties.Instrument = true
	module.Module.properties.Installable = proptools.BoolPtr(true)

	module.AddProperties(
		&module.Module.properties,
		&module.Module.deviceProperties,
		&module.Module.protoProperties,
		&module.aaptProperties,
		&module.appProperties,
		&module.appTestProperties,
		&module.testProperties)

	InitJavaModule(module, android.DeviceSupported)
	return module
}
