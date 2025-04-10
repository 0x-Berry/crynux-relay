package stats

import (
	"context"
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetNodeEventLogs(ctx *gin.Context) error {
	start := time.Now().Add(-time.Hour)

	var allTasks []models.InferenceTask

	limit := 500
	offset := 0
	for {
		var tasks []models.InferenceTask
		err := func() error {
			dbCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
			defer cancel()
			return config.GetDB().WithContext(dbCtx).Model(&models.InferenceTask{}).
				Where("created_at >= ?", start).
				Where("start_time IS NOT NULL").
				Order("id").Offset(offset).Limit(limit).Find(&tasks).Error
		}()
		if err != nil {
			return response.NewExceptionResponse(err)
		}
		allTasks = append(allTasks, tasks...)

		if len(tasks) < limit {
			break
		}
		offset += limit
	}

	var nodeAddresses []string
	for _, task := range allTasks {
		nodeAddresses = append(nodeAddresses, task.SelectedNode)
	}

	var nodeDatas []models.NetworkNodeData

	if err := func() error {
		dbCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
		defer cancel()
		return config.GetDB().WithContext(dbCtx).Model(&models.NetworkNodeData{}).Where("address IN (?)", nodeAddresses).Find(&nodeDatas).Error
	}(); err != nil {
		return response.NewExceptionResponse(err)
	}

	nodeDataMap := make(map[string]models.NetworkNodeData)
	for _, nodeData := range nodeDatas {
		nodeDataMap[nodeData.Address] = nodeData
	}

	var builder strings.Builder
	for _, task := range allTasks {
		nodeData := nodeDataMap[task.SelectedNode]
		timeString := task.StartTime.Time.UTC().Format(time.RFC3339)
		fmt.Fprintf(&builder, "[%s] [%s] [%s.%d] [%s] [%s] [%s]\n", timeString, "Node selected", nodeData.CardModel, nodeData.VRam, task.SelectedNode, task.SelectedNode, task.TaskIDCommitment)
		if task.Status == models.TaskEndGroupRefund || task.Status == models.TaskEndAborted || task.Status == models.TaskEndInvalidated {
			timeString = task.ValidatedTime.Time.UTC().Format(time.RFC3339)
			fmt.Fprintf(&builder, "[%s] [%s] [%s.%d] [%s] [%s] [%s]\n", timeString, "Node released", nodeData.CardModel, nodeData.VRam, task.SelectedNode, task.SelectedNode, task.TaskIDCommitment)
		} else if task.Status == models.TaskEndSuccess {
			timeString = task.ResultUploadedTime.Time.UTC().Format(time.RFC3339)
			fmt.Fprintf(&builder, "[%s] [%s] [%s.%d] [%s] [%s] [%s]\n", timeString, "Node released", nodeData.CardModel, nodeData.VRam, task.SelectedNode, task.SelectedNode, task.TaskIDCommitment)
		}
	}

	ctx.String(http.StatusOK, builder.String())
	return nil
}
