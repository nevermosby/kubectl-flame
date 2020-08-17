//: Copyright Verizon Media
//: Licensed under the terms of the Apache 2.0 License. See LICENSE file in the project root for terms.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/VerizonMedia/kubectl-flame/cli/cmd/data"
	"github.com/VerizonMedia/kubectl-flame/cli/cmd/handler"
	"github.com/VerizonMedia/kubectl-flame/cli/cmd/kubernetes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func Flame(target *data.TargetDetails, configFlags *genericclioptions.ConfigFlags) {
	ns, err := kubernetes.Connect(configFlags)
	if err != nil {
		fmt.Printf("Failed connecting to kubernetes cluster: %v\n", err)
		os.Exit(1)
	}

	target.Namespace = ns
	ctx := context.Background()
	fmt.Print("Verifying target pod ... ")
	pod, err := kubernetes.GetPodDetails(target.PodName, target.Namespace, ctx)
	if err != nil {
		PrintError()
		fmt.Println(err.Error())
		os.Exit(1)
	}

	containerName, err := validatePod(pod, target.ContainerName)
	if err != nil {
		PrintError()
		fmt.Println(err.Error())
		os.Exit(1)
	}

	containerId, err := kubernetes.GetContainerId(containerName, pod)
	if err != nil {
		PrintError()
		fmt.Println(err.Error())
		os.Exit(1)
	}

	PrintSuccess()
	target.ContainerName = containerName
	target.ContainerId = containerId
	fmt.Print("Launching profiler ... ")
	profileId, job, err := kubernetes.LaunchFlameJob(pod, target, ctx)
	if err != nil {
		PrintError()
		fmt.Print(err.Error())
		os.Exit(1)
	}

	target.Id = profileId
	profilerPod, err := kubernetes.WaitForPodStart(target, ctx)
	if err != nil {
		PrintError()
		fmt.Println(err.Error())
		os.Exit(1)
	}

	PrintSuccess()
	apiHandler := &handler.ApiEventsHandler{
		Job:    job,
		Target: target,
	}
	done, err := kubernetes.GetLogsFromPod(profilerPod, apiHandler, ctx)
	if err != nil {
		PrintError()
		fmt.Println(err.Error())
	}

	<-done
}

func validatePod(pod *v1.Pod, specificContainer string) (string, error) {
	if pod == nil {
		return "", errors.New(fmt.Sprintf("Could not find pod %s in Namespace %s",
			targetDetails.PodName, targetDetails.Namespace))
	}

	if len(pod.Spec.Containers) != 1 {
		var containerNames []string
		for _, container := range pod.Spec.Containers {
			if container.Name == specificContainer {
				return container.Name, nil // Found given container
			}

			containerNames = append(containerNames, container.Name)
		}

		return "", errors.New(fmt.Sprintf("Could not determine container. please specify one of %v", containerNames))
	}

	return pod.Spec.Containers[0].Name, nil
}
