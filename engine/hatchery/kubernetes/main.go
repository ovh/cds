package main

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, _ := clientcmd.BuildConfigFromFlags("", "/Users/benjamincoenen/.kube/config")
	// creates the clientset
	clientset, _ := kubernetes.NewForConfig(config)
	// access the API to list pods
	pods, _ := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	for _, item := range pods.Items {
		fmt.Println(item.GetName())
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	fmt.Println("Creating pod...")
	var gracePeriodSecs int64
	// podSpec := apiv1.PodSpec{
	// 	RestartPolicy:                 apiv1.RestartPolicyNever,
	// 	TerminationGracePeriodSeconds: &gracePeriodSecs,
	// 	Containers: []apiv1.Container{
	// 		{
	// 			Name:  "worker",
	// 			Image: "ovhcom/cds-worker",
	// 			Env: []apiv1.EnvVar{
	// 				{Name: "CDS_API", Value: "http://192.168.1.5:8081"},
	// 				{Name: "CDS_NAME", Value: "K8S"},
	// 				{Name: "CDS_TOKEN", Value: "d05993ed96ae002908075cdaecf622549f51a6a9d5099728317679de455a7fc6"},
	// 				{Name: "CDS_SINGLE_USE", Value: "1"},
	// 			},
	// 		},
	// 	},
	// }
	//
	// var completions int32 = 1
	// job, err := clientset.BatchV1().Jobs(apiv1.NamespaceDefault).Create(&batchv1.Job{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name: "my-worker-pod",
	// 	},
	// 	Spec: batchv1.JobSpec{
	// 		Completions: &completions,
	// 		Template:    apiv1.PodTemplateSpec{Spec: podSpec},
	// 	},
	// })

	// fmt.Println(job)

	pod, err := clientset.CoreV1().Pods(apiv1.NamespaceDefault).Create(&apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-worker-pod2",
			DeletionGracePeriodSeconds: &gracePeriodSecs,
		},
		Spec: apiv1.PodSpec{
			RestartPolicy:                 apiv1.RestartPolicyNever,
			TerminationGracePeriodSeconds: &gracePeriodSecs,
			Containers: []apiv1.Container{
				{
					Name:  "worker",
					Image: "ovhcom/cds-worker",
					Env: []apiv1.EnvVar{
						{Name: "CDS_API", Value: "http://192.168.1.5:8081"},
						{Name: "CDS_NAME", Value: "K8S"},
						{Name: "CDS_TOKEN", Value: "d05993ed96ae002908075cdaecf622549f51a6a9d5099728317679de455a7fc6"},
						{Name: "CDS_SINGLE_USE", Value: "1"},
					},
				},
			},
		},
	})

	if err != nil {
		panic(err)
	}

	fmt.Println(pod)
}
