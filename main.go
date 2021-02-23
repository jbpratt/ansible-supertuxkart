package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"

	ansibler "github.com/apenella/go-ansible"
	"github.com/jbpratt/ansible-supertuxkart/internal/node"
)

const (
	prefix   = "ansible-supertuxkart"
	hostname = "stk-memes"
)

type cfg struct {
	IdentityFile string         `json:"identity_file"`
	OVHConfig    node.OVHConfig `json:"ovh_config"`
}

func main() {
	ctx := context.Background()

	cfgPath := flag.String("path", "config.json", "path to config file")
	ovhSKU := flag.String("sku", "B2-15", "OVH SKU")
	ovhRegion := flag.String("sku", "BHS5", "OVH Region")
	flag.Parse()

	logger := log.New(os.Stdout, prefix, log.LstdFlags)

	file, err := os.Open(*cfgPath)
	if err != nil {
		logger.Fatalln("failed to open cfg file:", cfgPath, err)
	}

	contents, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Fatalln("failed to read cfg file:", err)
	}

	config := &cfg{}
	if err = json.Unmarshal(contents, config); err != nil {
		logger.Fatalln("failed to unmarshal cfg contents:", err)
	}

	pubkeyFile, err := ioutil.ReadFile(config.IdentityFile + ".pub")
	if err != nil {
		logger.Fatalln("error reading ssh public key", err)
	}
	pubkey := string(bytes.Trim(pubkeyFile, "\r\n\t "))

	privkeyFile, err := ioutil.ReadFile(config.IdentityFile)
	if err != nil {
		logger.Fatalln("error reading ssh public key", err)
	}

	driver, err := node.NewOVHDriver(
		"CA",
		config.OVHConfig.AppKey,
		config.OVHConfig.AppSecret,
		config.OVHConfig.ConsumerKey,
		config.OVHConfig.ProjectID,
	)
	if err != nil {
		logger.Fatalln(err)
	}

	req := &node.CreateRequest{
		User:        driver.DefaultUser(),
		Name:        hostname,
		Region:      *ovhRegion,
		SKU:         *ovhSKU,
		SSHKey:      pubkey,
		BillingType: node.Hourly,
	}

	logger.Println("creating node")

	n, err := driver.Create(ctx, req)
	if err != nil {
		logger.Fatalln(req, err)
	}

	logger.Println("node created")

	logger.Println("sleeping 1m for node creation")
	time.Sleep(1 * time.Minute)

	playbook := &ansibler.AnsiblePlaybookCmd{
		ExecPrefix: prefix,
		Playbook:   "site.yml",
		Options: &ansibler.AnsiblePlaybookOptions{
			Inventory: n.Networks.V4[0],
		},
		ConnectionOptions: &ansibler.AnsiblePlaybookConnectionOptions{
			User:       driver.DefaultUser(),
			PrivateKey: string(privkeyFile),
		},
		PrivilegeEscalationOptions: &ansibler.AnsiblePlaybookPrivilegeEscalationOptions{
			Become: true,
		},
	}

	if err = playbook.Run(); err != nil {
		logger.Fatalln(err)
	}
}
