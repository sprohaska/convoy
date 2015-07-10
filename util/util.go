package util

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/mcuadros/go-version"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	PRESERVED_CHECKSUM_LENGTH = 64
)

func LoadConfig(path, name string, v interface{}) error {
	fileName := filepath.Join(path, name)
	st, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	file, err := os.Open(fileName)
	if err != nil {
		return err
	}

	defer file.Close()

	data := make([]byte, st.Size())
	_, err = file.Read(data)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, v); err != nil {
		return err
	}
	return nil
}

func SaveConfig(path, name string, v interface{}) error {
	fileName := filepath.Join(path, name)
	j, err := json.Marshal(v)
	if err != nil {
		return err
	}

	tmpFileName := filepath.Join(path, name+".tmp")

	f, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write(j); err != nil {
		return err
	}

	if _, err = os.Stat(fileName); err == nil {
		err = os.Remove(fileName)
		if err != nil {
			return err
		}
	}

	if err := os.Rename(tmpFileName, fileName); err != nil {
		return err
	}

	return nil
}

func EncodeData(v interface{}) (*bytes.Buffer, error) {
	param := bytes.NewBuffer(nil)
	j, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if _, err := param.Write(j); err != nil {
		return nil, err
	}
	return param, nil
}

func ConfigExists(path, name string) bool {
	fileName := filepath.Join(path, name)
	_, err := os.Stat(fileName)
	return err == nil
}

func RemoveConfig(path, name string) error {
	fileName := filepath.Join(path, name)
	if _, err := Execute("rm", []string{"-f", fileName}); err != nil {
		return err
	}
	return nil
}

func ListConfigIDs(path, prefix, suffix string) []string {
	out, err := Execute("find", []string{path,
		"-maxdepth", "1",
		"-name", prefix + "*" + suffix,
		"-printf", "%f "})
	if err != nil {
		return []string{}
	}
	if len(out) == 0 {
		return []string{}
	}
	fileResult := strings.Split(strings.TrimSpace(string(out)), " ")
	result := make([]string, len(fileResult))
	for i := range fileResult {
		f := fileResult[i]
		f = strings.TrimPrefix(f, prefix)
		result[i] = strings.TrimSuffix(f, suffix)
	}
	return result
}

func MkdirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModeDir|0700); err != nil {
			return err
		}
	}
	return nil
}

func GetChecksum(data []byte) string {
	checksumBytes := sha512.Sum512(data)
	checksum := hex.EncodeToString(checksumBytes[:])[:PRESERVED_CHECKSUM_LENGTH]
	return checksum
}

func LockFile(fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

func UnlockFile(fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := unix.Flock(int(f.Fd()), unix.LOCK_UN); err != nil {
		return err
	}
	if _, err := Execute("rm", []string{fileName}); err != nil {
		return err
	}
	return nil
}

func SliceToMap(slices []string) map[string]string {
	result := map[string]string{}
	for _, v := range slices {
		pair := strings.Split(v, "=")
		if len(pair) != 2 {
			return nil
		}
		result[pair[0]] = pair[1]
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func GetFileChecksum(filePath string) (string, error) {
	output, err := Execute("sha512sum", []string{"-b", filePath})
	if err != nil {
		return "", err
	}
	return strings.Split(string(output), " ")[0], nil
}

func CompressFile(filePath string) error {
	if _, err := Execute("gzip", []string{filePath}); err != nil {
		return err
	}
	return nil
}

func UncompressFile(filePath string) error {
	if _, err := Execute("gunzip", []string{filePath}); err != nil {
		return err
	}
	return nil
}

func Copy(src, dst string) error {
	if _, err := Execute("cp", []string{src, dst}); err != nil {
		return err
	}
	return nil
}

func AttachLoopbackDevice(file string, readonly bool) (string, error) {
	params := []string{"-v", "-f"}
	if readonly {
		params = append(params, "-r")
	}
	params = append(params, file)
	out, err := Execute("losetup", params)
	if err != nil {
		return "", err
	}
	dev := strings.TrimSpace(strings.SplitAfter(string(out[:]), "device is")[1])
	return dev, nil
}

func DetachLoopbackDevice(file, dev string) error {
	output, err := Execute("losetup", []string{dev})
	if err != nil {
		return err
	}
	out := strings.TrimSpace(string(output))
	suffix := "(" + file + ")"
	if !strings.HasSuffix(out, suffix) {
		return fmt.Errorf("Unmatched source file, output %v, suffix %v", out, suffix)
	}
	if _, err := Execute("losetup", []string{"-d", dev}); err != nil {
		return err
	}
	return nil
}

func ValidateUUID(s string) bool {
	return uuid.Parse(s) != nil
}

func ValidateName(name string) bool {
	validName := regexp.MustCompile(`^[0-9a-z_.]+$`)
	return validName.MatchString(name)
}

func ParseSize(size string) (int64, error) {
	size = strings.ToLower(size)
	readableSize := regexp.MustCompile(`^[0-9.]+[kmg]$`)
	if !readableSize.MatchString(size) {
		value, err := strconv.ParseInt(size, 10, 64)
		if value == 0 && err == nil {
			err = fmt.Errorf("Invalid size %v", size)
		}
		return value, err
	}

	last := len(size) - 1
	unit := string(size[last])
	value, err := strconv.ParseInt(size[:last], 10, 64)
	if err != nil {
		return 0, err
	}

	kb := int64(1024)
	mb := 1024 * kb
	gb := 1024 * mb
	switch unit {
	case "k":
		value *= kb
	case "m":
		value *= mb
	case "g":
		value *= gb
	default:
		return 0, fmt.Errorf("Unrecongized size value %v", size)
	}
	if value == 0 && err == nil {
		err = fmt.Errorf("Invalid size %v", size)
	}
	return value, err
}

func CheckBinaryVersion(binaryName, minVersion string, args []string) error {
	output, err := exec.Command(binaryName, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed version check for %s, due to %s", binaryName, err.Error())
	}
	v := strings.TrimSpace(string(output))
	if version.Compare(v, minVersion, "<") {
		return fmt.Errorf("Minimal require version for %s is %s, detected %s",
			binaryName, minVersion, v)
	}
	return nil
}

func Execute(binary string, args []string) (string, error) {
	output, err := exec.Command(binary, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Failed to execute: %v %v, output %v, error %v", binary, args, string(output), err)
	}
	return string(output), nil
}

func Now() string {
	return time.Now().Format(time.RubyDate)
}

func AddToIndex(key, value string, index map[string]string) error {
	if oldValue, exists := index[key]; exists {
		if oldValue != value {
			return fmt.Errorf("BUG: Conflict when updating index, %v was mapped to %v, but %v want to be mapped too", key, oldValue, value)
		}
		return nil
	}
	index[key] = value
	return nil
}

func RemoveFromIndex(key string, index map[string]string) error {
	if _, exists := index[key]; !exists {
		return fmt.Errorf("BUG: About to remove %v from index, but it didn't exist in it", key)
	}
	delete(index, key)
	return nil
}
