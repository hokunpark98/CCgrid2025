package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
	"traffic-scheduler/graphGenerator"
	"traffic-scheduler/metricCollector"
	"traffic-scheduler/trafficAllocator"
)

func LogComponentGraph(componentGraph *graphGenerator.ComponentGraph, logFile *os.File) {
	logFile.WriteString("Component Graph:\n")
	log.Print("Component Graph:")
	logFile.WriteString("  Components:\n")
	log.Print("  Components:")

	for component := range componentGraph.Components {
		logFile.WriteString("    - " + component + "\n")
		log.Print("    - " + component)

		logFile.WriteString("  Links:\n")
		log.Print("  Links:")
		for _, link := range componentGraph.Components[component] {
			logFile.WriteString("    - [" + component + " -> " + link + "]\n")
			log.Print("    - [" + component + " -> " + link + "]")
		}
		logFile.WriteString(fmt.Sprintf("\n"))
		log.Print("")
	}
}

func LogComponentPodMap(componentPodMap *metricCollector.ComponentPodMap, logFile *os.File) {
	logFile.WriteString("Component to Pod Map:\n")
	log.Print("Component to Pod Map:\n")
	for component, pods := range componentPodMap.Components {
		logFile.WriteString("  Comonent:" + component + "\n")
		log.Print("  Comonent:" + component + "\n")

		for _, pod := range pods {
			logFile.WriteString("    [" + pod.PodName + ", " + pod.PodIP + ", " +
				pod.HostName + ", " + pod.HostIP + "]\n")
			log.Print("    [" + pod.PodName + ", " + pod.PodIP + ", " + "[" + strings.Join(pod.Ports, ", ") + "]" + ", " +
				pod.HostName + ", " + pod.HostIP + "]\n")
		}
	}
	logFile.WriteString(fmt.Sprintf("\n"))
	log.Print("")
}

func LogRequestCountPerPod(requestCountMap *metricCollector.RequestCountMap, logFile *os.File) {
	logFile.WriteString(fmt.Sprintf("Request Count Per Pod:\n"))
	log.Print("Request Count Per Pod:")
	for _, component := range requestCountMap.Components {
		logFile.WriteString(fmt.Sprintf("  Component: %s\n", component.ComponentName))
		log.Print(fmt.Sprintf("  Component: %s", component.ComponentName))
		for _, podRequest := range component.PodRequestMap {
			logFile.WriteString(fmt.Sprintf("    Pod: %s, Request Count: %d\n", podRequest.PodName, podRequest.RequestCount))
			log.Print(fmt.Sprintf("    Pod: %s, Request Count: %d", podRequest.PodName, podRequest.RequestCount))
		}
	}
	logFile.WriteString(fmt.Sprintf("\n"))
	log.Print("")
}

func LogRequestDurationData(requestDurationMap *metricCollector.RequestDurationMap, logFile *os.File) {
	logFile.WriteString(fmt.Sprintf("Request Duration Per Pod:\n"))
	log.Print("Request Duration Per Pod:")
	for _, component := range requestDurationMap.Components {
		logFile.WriteString(fmt.Sprintf("  Component: %s\n", component.ComponentName))
		log.Print(fmt.Sprintf("  Component: %s", component.ComponentName))
		for _, podDuration := range component.PodDurations {
			logFile.WriteString(fmt.Sprintf("    Pod: %s, Request Duration: %d ms\n", podDuration.PodName, podDuration.RequestDuration))
			log.Print(fmt.Sprintf("    Pod: %s, Request Duration: %d ms", podDuration.PodName, podDuration.RequestDuration))
		}
	}
	logFile.WriteString(fmt.Sprintf("\n"))
	log.Print("")
}

func LogCpuUtilizationPerPod(cpuUtilizationMap *metricCollector.CpuUtilizationMap, logFile *os.File) {
	logFile.WriteString(fmt.Sprintf("CPU Utilization Per Pod:\n"))
	log.Print("CPU Utilization Per Pod:")
	for _, component := range cpuUtilizationMap.Components {
		logFile.WriteString(fmt.Sprintf("  Component: %s\n", component.ComponentName))
		log.Print(fmt.Sprintf("  Component: %s", component.ComponentName))
		for _, podCpu := range component.PodCpuUsage {
			logFile.WriteString(fmt.Sprintf("    Pod: %s, CPU Utilization: %d%%\n", podCpu.PodName, podCpu.CpuUtilization))
			log.Print(fmt.Sprintf("    Pod: %s, CPU Utilization: %d%%", podCpu.PodName, podCpu.CpuUtilization))
		}
	}
	logFile.WriteString(fmt.Sprintf("\n"))
	log.Print("")
}

func LogNodeCpuHz(nodeCPUFrequencyMap *metricCollector.NodeCpuHzMap, logFile *os.File) {
	logFile.WriteString("Node CPU Hz:\n")
	log.Print("Node CPU Hz:")
	for node, nodeFreq := range nodeCPUFrequencyMap.Nodes {
		logFile.WriteString(fmt.Sprintf("  NodeName: %s, Hertz: %d\n", node, nodeFreq.Hertz))
		log.Print(fmt.Sprintf("  NodeName: %s, Hertz: %d", node, nodeFreq.Hertz))
	}
	logFile.WriteString(fmt.Sprintf("\n"))
	log.Print("")
}

func LogTrafficAllocationResult(trafficAllocationResult *trafficAllocator.ProportionMap, logFile *os.File) {
	logFile.WriteString("Traffic Allocation Result:\n")
	for sourceComponent, destinationMap := range trafficAllocationResult.Components {
		logFile.WriteString(fmt.Sprintf("  Source Component: %s\n", sourceComponent))
		for destinationComponent, sourcePodDataList := range destinationMap {
			logFile.WriteString(fmt.Sprintf("    Destination Component: %s\n", destinationComponent))
			for _, sourcePodData := range sourcePodDataList {
				logFile.WriteString(fmt.Sprintf("      Source Pod: %s\n", sourcePodData.SourcePodName))
				for _, proportionData := range sourcePodData.ProportionDatas {
					logFile.WriteString(fmt.Sprintf("        -> Destination Pod: %s, Proportion: %d\n", proportionData.DestinationPodName, proportionData.Proportion))
				}
			}
		}
	}
}
