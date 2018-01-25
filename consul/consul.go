package consul

import (
	"crypto/tls"
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"sort"
	"strings"
)

type ByLength []string

func (s ByLength) Len() int {
	return len(s)
}

func (s ByLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ByLength) Less(i, j int) bool {
	return len(s[i]) > len(s[j])
}

func getConsul() *consulapi.Client {
	addr := viper.GetString("addr")
	token := viper.GetString("token")
	auth := viper.GetString("auth")
	ssl := viper.GetString("ssl")

	verbose := viper.GetBool("verbose")

	if verbose {
		fmt.Printf("Connecting to %s %s %s %s\n", addr, token, auth, ssl)
	}
	config := consulapi.DefaultConfig()
	config.Address = addr

	if ssl == "true" {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		config.HttpClient = &http.Client{Transport: transport}
		config.Scheme = "https"
	} else {
		config.Scheme = "http"
	}

	if auth != "" {
		sliceAuth := strings.Split(auth, ":")
		if len(sliceAuth) != 2 {
			fmt.Fprintln(os.Stderr, "Invalid AUTH string specified.")
			os.Exit(132)
		}
		user := sliceAuth[0]
		pass := sliceAuth[1]
		config.HttpAuth = &consulapi.HttpBasicAuth{Username: user, Password: pass}
	}

	if token != "" {
		config.Token = token
	}

	consul, _ := consulapi.NewClient(config)
	return consul
}

func pathIsUnique(s []string, path string) bool {
	for _, p := range s {
		if p != path && strings.HasPrefix(path, p) {
			return false
		}
	}
	return true
}

func pathsToQuery(paths []string) []string {
	sort.Sort(ByLength(paths))

	var uniquePaths []string

	for _, path := range paths {
		path = strings.Trim(path, "/")
		if pathIsUnique(paths, path) {
			uniquePaths = append(uniquePaths, path)
		}
	}

	return uniquePaths
}

func processEnv(envMap map[string]map[string]string, paths []string) map[string]string {
	export := viper.GetBool("export")

	sort.Sort(sort.Reverse(ByLength(paths)))

	env := make(map[string]string)

	for _, path := range paths {
		path = strings.Trim(path, "/")
		if _, ok := envMap[path]; ok {
			for k, v := range envMap[path] {
				if _, ok := env[k]; !ok {
					env[k] = v
				}
			}
		}
	}
	for k, v := range env {
		if export {
			fmt.Printf("export %s='%s'\n", k, v)
		} else {
			fmt.Printf("%s=%s\n", k, v)
		}
	}
	return env
}

func Get() {
	paths := viper.GetStringSlice("path")
	verbose := viper.GetBool("verbose")

	consul := getConsul()

	uniquePaths := pathsToQuery(paths)

	kv := consul.KV()

	envMap := map[string]map[string]string{}

	for _, p := range uniquePaths {
		if verbose {
			fmt.Fprintln(os.Stderr, "Looking at", p)
		}
		kvPairs, qm, err := kv.List(p, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err, qm)
			os.Exit(133)
		} else {
			for _, kvPair := range kvPairs {
				val := string(kvPair.Value)

				parts := strings.Split(kvPair.Key, "/")
				folder := strings.Join(parts[:len(parts)-1], "/")
				folder = strings.Trim(folder, "/")
				varName := parts[len(parts)-1]

				if val != "" {
					if _, ok := envMap[folder]; !ok {
						envMap[folder] = make(map[string]string)
					}
					envMap[folder][varName] = val
				}
			}
		}
	}

	processEnv(envMap, paths)
}