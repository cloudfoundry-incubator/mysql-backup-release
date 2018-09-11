package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"streaming-mysql-backup-tool/config"
)

func saveScript(scriptContents string) string {
	tmpFile := createTmpFile("testScript")

	_, err := tmpFile.WriteString(scriptContents)
	Expect(err).ShouldNot(HaveOccurred())

	err = tmpFile.Close()
	Expect(err).ShouldNot(HaveOccurred())

	filePath, err := filepath.Abs(tmpFile.Name())
	Expect(err).ShouldNot(HaveOccurred())

	return filePath
}

func saveBashScript(scriptContents string) string {
	scriptPath := saveScript("#!/bin/bash\n" + scriptContents)

	return "bash " + scriptPath
}

var _ = Describe("Main", func() {

	var (
		session              *gexec.Session
		backupUrl            string
		command              *exec.Cmd
		request              *http.Request
		httpClient           *http.Client
		rootConfig           config.Config
		expectedResponseBody= "my_output"
	)

	BeforeEach(func() {
		rootConfig = config.Config{
			Port:    int(49000 + GinkgoParallelNode()),
			Command: fmt.Sprintf("echo -n %s", expectedResponseBody),
			PidFile: tmpFilePath("pid"),
			Credentials: config.Credentials{
				Username: "username",
				Password: "password",
			},
		}
	})

	JustBeforeEach(func() {
		// In case individual tests want to modify their rootConfig variable after BeforeEach
		writeConfig(rootConfig)

		backupUrl = fmt.Sprintf("http://localhost:%d/backup", rootConfig.Port)
		var (
			err error
		)
		request, err = http.NewRequest("GET", backupUrl, nil)
		request.SetBasicAuth("username", "password")

		httpClient = &http.Client{}
		command = exec.Command(pathToMainBinary, fmt.Sprintf("-configPath=%s", configPath))
		session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() error {
			_, err := http.Get(backupUrl)
			return err
		}).Should(Succeed())
	})

	AfterEach(func() {
		session.Kill()
		session.Wait()
		err := os.Remove(rootConfig.PidFile)
		Expect(err).ToNot(HaveOccurred())
	})
	

	Describe("Writing PID file", func() {
		var (
			pidFilePath string
		)
		BeforeEach(func() {
			pidFilePath = rootConfig.PidFile
		})

		It("Writes its PID file to the location specified ", func() {
			Expect(pidFilePath).To(BeAnExistingFile())
		})

		It("Checks whether the PID file content matches the process ID", func() {
			fileBytes, err := ioutil.ReadFile(rootConfig.PidFile)
			Expect(err).ToNot(HaveOccurred())
			actualPid, err := strconv.Atoi(string(fileBytes))
			Expect(err).ToNot(HaveOccurred())
			Expect(actualPid).To(Equal(command.Process.Pid))
		})
	})

	Describe("Initiating a backup", func() {
		It("Returns status 200 when the backup is scheduled", func() {
			resp, err := httpClient.Do(request)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
		})

		It("Returns the output from the configured backup command as the response body", func() {
			resp, err := httpClient.Do(request)
			Expect(err).ShouldNot(HaveOccurred())

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(body).To(Equal([]byte(expectedResponseBody)))
		})

		It("Has a trailer with empty Error field if it succeeded", func() {
			resp, err := httpClient.Do(request)
			Expect(err).ShouldNot(HaveOccurred())

			_, err = ioutil.ReadAll(resp.Body)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(resp.Trailer.Get(http.CanonicalHeaderKey("X-Backup-Error"))).To(BeEmpty())
		})

		Context("when the backup is unsuccessful", func() {
			BeforeEach(func() {
				rootConfig.Command = "cat nonexistentfile"
			})

			It("has HTTP 200 status code but writes the error to the trailer", func() {
				resp, err := httpClient.Do(request)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(200))

				// NOTE: You must read the body from the response in order to populate the response's
				// trailers
				body, err := ioutil.ReadAll(resp.Body)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(body).To(Equal([]byte("")))

				Expect(resp.Trailer.Get(http.CanonicalHeaderKey("X-Backup-Error"))).To(ContainSubstring("exit status 1"))
			})
		})

		Context("when the command fails halfway through", func() {
			BeforeEach(func() {
				// https://www.percona.com/doc/percona-xtrabackup/2.1/xtrabackup_bin/xtrabackup_exit_codes.html
				longRunningScript := `echo -n hello
										exit 1
										echo world`

				rootConfig.Command = saveBashScript(longRunningScript)
			})

			It("has HTTP 200 status code but writes the error to the trailer", func() {
				resp, err := httpClient.Do(request)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(200))

				// NOTE: You must read the body from the response in order to populate the response's
				// trailers
				body, err := ioutil.ReadAll(resp.Body)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(body).To(Equal([]byte("hello")))

				Expect(resp.Trailer.Get(http.CanonicalHeaderKey("X-Backup-Error"))).To(ContainSubstring("exit status 1"))
			})
		})
	})

	Describe("REGRESSION: Hitting the same endpoint twice", func() {
		It("does not fail", func() {
			resp, err := httpClient.Do(request)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			resp, err = httpClient.Do(request)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
		})
	})

	Describe("basic auth credentials", func() {
		It("expects Basic Auth creds", func() {
			resp, err := httpClient.Get(backupUrl)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			Expect(resp.Header.Get("WWW-Authenticate")).To(Equal(`Basic realm="Authorization Required"`))

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(body).ToNot(ContainSubstring(expectedResponseBody))
		})

		It("accepts good Basic Auth creds", func() {
			req, err := http.NewRequest("GET", backupUrl, nil)
			req.SetBasicAuth("username", "password")
			resp, err := httpClient.Do(req)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("does not accept bad Basic Auth creds", func() {
			req, err := http.NewRequest("GET", backupUrl, nil)
			req.SetBasicAuth("bad_username", "bad_password")
			resp, err := httpClient.Do(req)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(body).ToNot(ContainSubstring(expectedResponseBody))
		})
	})


})
