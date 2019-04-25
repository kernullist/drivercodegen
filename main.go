//
// prerequisite:
// go get golang.org/x/sys/windows/registry
// go get -u github.com/gonutz/w32
// go get github.com/google/uuid
//
// build:
// go build -o drivercodegen.exe main.go projecttemplate.go codetemplate.go
//
// usage : 
// codetemplate.go -name MyDriver -path d:\codebase

package main

import (
	"golang.org/x/sys/windows/registry"
	"github.com/gonutz/w32"
	"github.com/google/uuid"
	"path/filepath"
	"log"
	"fmt"
	"flag"
	"os"
	"strings"
)

const (
	REG_UNINSTALL_WOW64_PATH = `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`
	VISUAL_STUDIO_2019_PRO = `Visual Studio Professional 2019`
	DISPLAY_NAME = `DisplayName`
	INSTALL_LOCATION = `InstallLocation`
	SUBPATH_DEVENV = `Common7\IDE\devenv.exe`
	EXE_NAME = `MyApp`
	COMMON_NAME = `Common`
)

const (
	MARK_VSVERSION = `$VSVERSION$`
	MARK_PROJECTNAME_SYS = `$PROJECTNAME_SYS$`
	MARK_GUID_SOLUTION = `$GUID_SOLUTION$`
	MARK_GUID_SYS = `$GUID_SYS$`
	MARK_GUID_EXE = `$GUID_EXE$`
	MARK_GUID_RANDOM = `$GUID_RANDOM$`
)

var (
	solutionName string
	outputBasePath string
	outputPath string
	solutionFilePath string
	sysVcxprojFilePath string
	exeVcxprojFilePath string
	sysVcxprojFilterFilePath string
	exeVcxprojFilterFilePath string
	sysHeaderFilePath string
	sysCppFilePath string
	exeCppFilePath string
	commonFilePath string
	sysGuid string
	exeGuid string
)

func getRegStringValue(k registry.Key, path, name string) (string, error) {
	openedKey, err := registry.OpenKey(k, path, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer openedKey.Close()

	v, _, err := openedKey.GetStringValue(name)
	if err != nil {
		return "", err
	}

	return v, nil
}

func getVisualStudioInstallLocationPath() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, REG_UNINSTALL_WOW64_PATH, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return ""
	}
	defer k.Close()

	subNames, err := k.ReadSubKeyNames(-1)
	if err != nil {
    	return ""
	}

	for _, name := range subNames {
		displayName, _ := getRegStringValue(k, name, DISPLAY_NAME)
		if displayName == VISUAL_STUDIO_2019_PRO {
			installLocation, _ := getRegStringValue(k, name, INSTALL_LOCATION)
			if installLocation != "" {
				return filepath.Join(installLocation, SUBPATH_DEVENV)
			}
		}
	}

	return ""
}

func getFileVersion(filePath string) (versionString string) {
    size := w32.GetFileVersionInfoSize(filePath)
    if size <= 0 {
        return
    }

    info := make([]byte, size)
    ok := w32.GetFileVersionInfo(filePath, info)
    if !ok {
        return
    }

    fixed, ok := w32.VerQueryValueRoot(info)
    if !ok {
        return
	}
	
	version := fixed.FileVersion()
	versionString = fmt.Sprintf("%d.%d.%d.%d", 
		(version & 0xFFFF000000000000) >> 48,
		(version & 0x0000FFFF00000000) >> 32,
		(version & 0x00000000FFFF0000) >> 16,
		(version & 0x000000000000FFFF) >> 0)
	
	return
}

func makeFile(filePath, contents string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	buffer := []byte(contents)
	if _, err := f.Write(buffer); err != nil {
		return err
	}

	return nil
}

func replaceContents(contents, fromString, toString string) string {
	result := strings.Replace(contents, fromString, toString, -1)
	return result
}

func prepareDirectories() error {
	outputPath = filepath.Join(outputBasePath, solutionName)
	if _, err := os.Stat(outputPath); err == nil {
		log.Printf("[-] %s Already Exsits...\n", outputPath)
		return err
	}

	if err := os.Mkdir(outputPath, 0755); err != nil {
		log.Printf("[-] Failed to mkdir %s\n", outputPath)
		return err
	}

	sysPath := filepath.Join(outputPath, solutionName)
	if _, err := os.Stat(sysPath); err == nil {
		log.Printf("[-] %s Already Exsits...\n", sysPath)
		return err
	}

	if err := os.Mkdir(sysPath, 0755); err != nil {
		log.Printf("[-] Failed to mkdir %s\n", sysPath)
		return err
	}

	exePath := filepath.Join(outputPath, EXE_NAME)
	if _, err := os.Stat(exePath); err == nil {
		log.Printf("[-] %s Already Exsits...\n", exePath)
		return err
	}

	if err := os.Mkdir(exePath, 0755); err != nil {
		log.Printf("[-] Failed to mkdir %s\n", exePath)
		return err
	}

	commPath := filepath.Join(outputPath, COMMON_NAME)
	if _, err := os.Stat(commPath); err == nil {
		log.Printf("[-] %s Already Exsits...\n", commPath)
		return err
	}

	if err := os.Mkdir(commPath, 0755); err != nil {
		log.Printf("[-] Failed to mkdir %s\n", commPath)
		return err
	}

	solutionFilePath = filepath.Join(outputPath, solutionName + `.sln`)
	sysVcxprojFilePath = filepath.Join(sysPath, solutionName + `.vcxproj`)
	exeVcxprojFilePath = filepath.Join(exePath, EXE_NAME + `.vcxproj`)
	sysVcxprojFilterFilePath = filepath.Join(sysPath, solutionName + `.vcxproj.filters`)
	exeVcxprojFilterFilePath = filepath.Join(exePath, EXE_NAME + `.vcxproj.filters`)
	commonFilePath = filepath.Join(commPath, COMMON_NAME + `.h`)

	sysHeaderFilePath = filepath.Join(sysPath, solutionName + `.h`)
	sysCppFilePath = filepath.Join(sysPath, solutionName + `.cpp`)
	exeCppFilePath = filepath.Join(exePath, EXE_NAME + `.cpp`)
	
	return nil
}

func genGuid() string {
	id, err := uuid.NewUUID()
    if err != nil {
        return ""
	}
	guid := fmt.Sprintf(`{%s}`, strings.ToUpper(id.String()))
	return guid
}

func makeSolutionFile(vsVersion string) error {
	sysGuid = genGuid()
	exeGuid = genGuid()
	solutionGuid := genGuid()

	contents := replaceContents(SOLUTION_TEMPLATE, MARK_VSVERSION, vsVersion)
	contents = replaceContents(contents, MARK_PROJECTNAME_SYS, solutionName)
	contents = replaceContents(contents, MARK_GUID_SOLUTION, solutionGuid)
	contents = replaceContents(contents, MARK_GUID_SYS, sysGuid)
	contents = replaceContents(contents, MARK_GUID_EXE, exeGuid)

	if err := makeFile(solutionFilePath, contents); err != nil {
		return err
	}
	return nil
}

func makeSysVcxprojFile() error {
	contents := replaceContents(VCXPROJ_SYS_TEMPLATE, MARK_GUID_SYS, sysGuid)
	contents = replaceContents(contents, MARK_PROJECTNAME_SYS, solutionName)
	if err := makeFile(sysVcxprojFilePath, contents); err != nil {
		return err
	}
	return nil
}

func makeExeVcxprojFile() error {
	contents := replaceContents(VCXPROJ_EXE_TEMPLATE, MARK_GUID_EXE, exeGuid)
	if err := makeFile(exeVcxprojFilePath, contents); err != nil {
		return err
	}
	return nil
}

func makeSysVcxprojFilterFile() error {
	guid := genGuid()
	contents := replaceContents(VCXPROJFILTER_SYS_TEMPLATE, MARK_GUID_RANDOM, guid)
	contents = replaceContents(contents , MARK_PROJECTNAME_SYS, solutionName)
	if err := makeFile(sysVcxprojFilterFilePath, contents); err != nil {
		return err
	}
	return nil
}

func makeExeVcxprojFilterFile() error {
	guid := genGuid()
	contents := replaceContents(VCXPROJFILTER_EXE_TEMPLATE, MARK_GUID_RANDOM, guid)
	if err := makeFile(exeVcxprojFilterFilePath, contents); err != nil {
		return err
	}
	return nil
}

func makeCodeFiles() error {
	contents_sys_h := replaceContents(DRIVER_HEADER_TEMPLATE, MARK_PROJECTNAME_SYS, solutionName)
	if err := makeFile(sysHeaderFilePath, contents_sys_h); err != nil {
		return err
	}

	contents_sys_cpp := replaceContents(DRIVER_CPP_TEMPLATE, MARK_PROJECTNAME_SYS, solutionName)
	if err := makeFile(sysCppFilePath, contents_sys_cpp); err != nil {
		return err
	}

	contents_exe_cpp := replaceContents(EXE_CPP_TEMPLATE, MARK_PROJECTNAME_SYS, solutionName)
	if err := makeFile(exeCppFilePath, contents_exe_cpp); err != nil {
		return err
	}

	contents_common_h := replaceContents(COMMON_HEADER_TEMPLATE, MARK_PROJECTNAME_SYS, solutionName)
	if err := makeFile(commonFilePath, contents_common_h); err != nil {
		return err
	}

	return nil
}

func main() {
	flag.StringVar(&solutionName, "name", "", "solution name")
	flag.StringVar(&outputBasePath, "path", "", "output base path")
	
	flag.Parse()
	if solutionName == "" || outputBasePath == "" {
		log.Println("[-] Invalid Parameter...")
		log.Println("[-] ex) drivercodegen.exe -name [solution name] -path [output base path]")
		return
	}

	vs2019Path := getVisualStudioInstallLocationPath()
	if vs2019Path == "" {
		log.Println("[-] Not found Visual Studio 2019....")
		return
	}
	log.Println("[+] Visual Studio 2019 Path : ", vs2019Path)

	vsVersion := getFileVersion(vs2019Path)
	if vsVersion == "" {
		log.Println("[-] Failed to get Visual Studio 2019 Version....")
		return
	}
	log.Println("[+] Visual Studio 2019 Version : ", vsVersion)

	//log.Println(SOLUTION_TEMPLATE)
	if err := prepareDirectories(); err != nil {
		log.Println("[-] Failed to prepareDirectories....")
		return
	}

	if err := makeSolutionFile(vsVersion); err != nil {
		log.Println("[-] Failed to makeSolutionFile....")
		return
	}

	if err := makeSysVcxprojFile(); err != nil {
		log.Println("[-] Failed to makeSysVcxprojFile....")
		return
	}

	if err := makeExeVcxprojFile(); err != nil {
		log.Println("[-] Failed to makeExeVcxprojFile....")
		return
	}

	if err := makeSysVcxprojFilterFile(); err != nil {
		log.Println("[-] Failed to makeSysVcxprojFilterFile....")
		return
	}

	if err := makeExeVcxprojFilterFile(); err != nil {
		log.Println("[-] Failed to makeExeVcxprojFilterFile....")
		return
	}

	if err := makeCodeFiles(); err != nil {
		log.Println("[-] Failed to makeSysCodeFile....")
		return
	}

	log.Printf("[+] Generated... Check %s\n", outputPath)
}