package name

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"streaming-mysql-backup-client/config"
)

type ArtifactNamer interface {
	Name(string, int) (string, error)
}

type artifactNamer struct {
	config config.Config
}

func NewArtifactNamer(config config.Config) artifactNamer {
	return artifactNamer{
		config: config,
	}
}

type metadata struct {
	IsFollower bool `json:"is_follower"`
}

func (n artifactNamer) Name(ip string, index int) (string, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: n.config.TLS.Config,
		},
	}

	req, _ := http.NewRequest("GET", ip + "/metadata", nil)
	// handle error
	req.SetBasicAuth(n.config.Credentials.Username, n.config.Credentials.Password)


	res, err := httpClient.Do(req)
	if err != nil {
		//this.logger.Error("Failed to make http request", err)
		return "failed to make http req", err //errors.WithStack(err)
	}
	if res.StatusCode != http.StatusOK {
		// error
		return "", errors.New(res.Status)
	}

	var md metadata
	if err := json.NewDecoder(res.Body).Decode(&md); err != nil {
		return "", err
	}

	baseArtifactName := fmt.Sprintf("mysql-backup-%d-%d",
		time.Now().Unix(),
		index,
	)

	if md.IsFollower {
		baseArtifactName += "_follower"
	}

	return baseArtifactName, nil
}

func main() {

}
