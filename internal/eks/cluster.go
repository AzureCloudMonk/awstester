package eks

import (
	"bytes"
	"io/ioutil"
	"strings"
	"text/template"
)

// isEKSDeletedGoClient returns true if error from EKS API indicates that
// the EKS cluster has already been deleted.
func isEKSDeletedGoClient(err error) bool {
	if err == nil {
		return false
	}
	/*
	   https://docs.aws.amazon.com/eks/latest/APIReference/API_Cluster.html#AmazonEKS-Type-Cluster-status

	   CREATING
	   ACTIVE
	   DELETING
	   FAILED
	*/
	// ResourceNotFoundException: No cluster found for name: awstester-155468BC717E03B003\n\tstatus code: 404, request id: 1e3fe41c-b878-11e8-adca-b503e0ba731d
	return strings.Contains(err.Error(), "No cluster found for name: ")
}

const kubeConfigTempl = `---
apiVersion: v1
clusters:
- cluster:
    server: {{.ClusterEndpoint}}
    certificate-authority-data: {{.ClusterCA}}
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: aws
  name: aws
current-context: aws
kind: Config
preferences: {}
users:
- name: aws
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      command: aws-iam-authenticator
      args:
        - token
        - -i
        - {{.ClusterName}}

`

type kubeConfig struct {
	ClusterEndpoint string
	ClusterCA       string
	ClusterName     string
}

func writeKubeConfig(ep, ca, clusterName, p string) (err error) {
	kc := kubeConfig{
		ClusterEndpoint: ep,
		ClusterCA:       ca,
		ClusterName:     clusterName,
	}
	tpl := template.Must(template.New("kubeCfgTempl").Parse(kubeConfigTempl))
	buf := bytes.NewBuffer(nil)
	if err = tpl.Execute(buf, kc); err != nil {
		return err
	}
	return ioutil.WriteFile(p, buf.Bytes(), 0600)
}

/*
expect:

NAME                 TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
service/kubernetes   ClusterIP   10.100.0.1   <none>        443/TCP   1m
*/
func isKubernetesControlPlaneReadyKubectl(kubectlOutput string) bool {
	return strings.Contains(kubectlOutput, "service/kubernetes") && strings.Contains(kubectlOutput, "ClusterIP")
}
