package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox"
	"github.com/qiniu/go-sdk/v7/sandbox/apis"
)

func main() {
	apiKey := os.Getenv("Qiniu_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("E2B_API_KEY")
	}
	if apiKey == "" {
		log.Fatal("请设置 Qiniu_API_KEY 或 E2B_API_KEY 环境变量")
	}

	apiURL := os.Getenv("Qiniu_SANDBOX_API_URL")
	if apiURL == "" {
		apiURL = os.Getenv("E2B_API_URL")
	}

	c, err := sandbox.NewClient(&sandbox.Config{
		APIKey:   apiKey,
		Endpoint: apiURL,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. 列出所有模板
	fmt.Println("=== 列出模板 ===")
	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		log.Fatalf("列出模板失败: %v", err)
	}
	fmt.Printf("共 %d 个模板:\n", len(templates))
	for _, tmpl := range templates {
		aliases := "-"
		if len(tmpl.Aliases) > 0 {
			aliases = fmt.Sprintf("%v", tmpl.Aliases)
		}
		fmt.Printf("  - %s\n", tmpl.TemplateID)
		fmt.Printf("    别名: %s, 公开: %v\n", aliases, tmpl.Public)
		fmt.Printf("    CPU: %d 核, 内存: %d MB, 磁盘: %d MB, envd: %s\n",
			tmpl.CPUCount, tmpl.MemoryMB, tmpl.DiskSizeMB, tmpl.EnvdVersion)
		fmt.Printf("    构建状态: %s, 构建次数: %d, 使用次数: %d\n",
			tmpl.BuildStatus, tmpl.BuildCount, tmpl.SpawnCount)
		fmt.Printf("    创建时间: %s, 更新时间: %s\n",
			tmpl.CreatedAt.Format(time.RFC3339), tmpl.UpdatedAt.Format(time.RFC3339))
	}

	// 2. 获取单个模板详情
	if len(templates) > 0 {
		fmt.Println("\n=== 获取模板详情 ===")
		detail, err := c.GetTemplate(ctx, templates[0].TemplateID, nil)
		if err != nil {
			log.Fatalf("获取模板详情失败: %v", err)
		}
		fmt.Printf("模板: %s, 构建数: %d\n", detail.TemplateID, len(detail.Builds))
		for i, build := range detail.Builds {
			if i >= 3 {
				fmt.Printf("  ... 省略 %d 条\n", len(detail.Builds)-3)
				break
			}
			fmt.Printf("  - %s (状态: %s, CPU: %d, 内存: %d MB)\n",
				build.BuildID, build.Status, build.CPUCount, build.MemoryMB)
		}
	}

	// 3. 创建模板
	fmt.Println("\n=== 创建模板 ===")
	cpuCount := apis.CPUCount(2)
	memoryMB := apis.MemoryMB(512)
	templateName := "sdk-example-template"
	resp, err := c.CreateTemplate(ctx, apis.CreateTemplateV3JSONRequestBody{
		Name:     &templateName,
		CPUCount: &cpuCount,
		MemoryMB: &memoryMB,
	})
	if err != nil {
		log.Fatalf("创建模板失败: %v", err)
	}
	templateID := resp.TemplateID
	buildID := resp.BuildID
	fmt.Printf("模板已创建: %s (构建: %s)\n", templateID, buildID)

	// 确保测试结束时清理
	defer func() {
		fmt.Println("\n=== 删除模板 ===")
		if err := c.DeleteTemplate(context.Background(), templateID); err != nil {
			fmt.Printf("删除模板失败: %v\n", err)
		} else {
			fmt.Printf("模板 %s 已删除\n", templateID)
		}
	}()

	// 4. 更新模板
	fmt.Println("\n=== 更新模板 ===")
	public := true
	if err := c.UpdateTemplate(ctx, templateID, apis.UpdateTemplateJSONRequestBody{
		Public: &public,
	}); err != nil {
		log.Fatalf("更新模板失败: %v", err)
	}
	fmt.Println("模板已更新为公开")

	// 5. 获取构建状态
	fmt.Println("\n=== 获取构建状态 ===")
	buildInfo, err := c.GetTemplateBuildStatus(ctx, templateID, buildID, nil)
	if err != nil {
		fmt.Printf("获取构建状态失败: %v\n", err)
	} else {
		fmt.Printf("构建 %s: 状态=%s, 模板=%s\n", buildInfo.BuildID, buildInfo.Status, buildInfo.TemplateID)
	}

	// 6. 获取构建日志
	fmt.Println("\n=== 获取构建日志 ===")
	buildLogs, err := c.GetTemplateBuildLogs(ctx, templateID, buildID, nil)
	if err != nil {
		fmt.Printf("获取构建日志失败: %v\n", err)
	} else {
		fmt.Printf("构建日志 (%d 条):\n", len(buildLogs.Logs))
		for i, entry := range buildLogs.Logs {
			if i >= 5 {
				fmt.Printf("  ... 省略 %d 条\n", len(buildLogs.Logs)-5)
				break
			}
			step := "-"
			if entry.Step != nil {
				step = *entry.Step
			}
			fmt.Printf("  [%s] [%s] %s: %s\n",
				entry.Timestamp.Format(time.RFC3339), entry.Level, step, entry.Message)
		}
	}

	// 7. 管理模板标签（ManageTemplateTags）
	fmt.Println("\n=== 管理模板标签 ===")
	tagResult, err := c.ManageTemplateTags(ctx, apis.ManageTemplateTagsJSONRequestBody{
		Target: fmt.Sprintf("%s:%s", templateName, "v1"),
		Tags:   []string{"latest", "stable"},
	})
	if err != nil {
		fmt.Printf("管理模板标签失败: %v\n", err)
	} else {
		fmt.Printf("标签已分配, 构建: %s, 标签: %v\n", tagResult.BuildID, tagResult.Tags)
	}

	// 8. 删除模板标签（DeleteTemplateTags）
	fmt.Println("\n=== 删除模板标签 ===")
	if err := c.DeleteTemplateTags(ctx, apis.DeleteTemplateTagsJSONRequestBody{
		Name: templateName,
		Tags: []string{"stable"},
	}); err != nil {
		fmt.Printf("删除模板标签失败: %v\n", err)
	} else {
		fmt.Println("标签 'stable' 已删除")
	}

	// 9. 通过别名查找模板（GetTemplateByAlias）
	fmt.Println("\n=== 通过别名查找模板 ===")
	if len(templates) > 0 && len(templates[0].Aliases) > 0 {
		alias := templates[0].Aliases[0]
		aliasResp, err := c.GetTemplateByAlias(ctx, alias)
		if err != nil {
			fmt.Printf("通过别名查找模板失败: %v\n", err)
		} else {
			fmt.Printf("别名 '%s' -> 模板: %s (公开: %v)\n", alias, aliasResp.TemplateID, aliasResp.Public)
		}
	} else {
		fmt.Println("没有可用的模板别名，跳过")
	}

	// 10. 获取模板文件上传链接（GetTemplateFiles）
	fmt.Println("\n=== 获取模板文件上传链接 ===")
	fileUpload, err := c.GetTemplateFiles(ctx, templateID, "example-hash-value")
	if err != nil {
		fmt.Printf("获取文件上传链接失败: %v\n", err)
	} else {
		url := "-"
		if fileUpload.URL != nil {
			url = *fileUpload.URL
		}
		fmt.Printf("文件是否已存在: %v, 上传地址: %s\n", fileUpload.Present, url)
	}

	// 11. 启动模板构建（StartTemplateBuild）
	fmt.Println("\n=== 启动模板构建 ===")
	fromImage := "ubuntu:latest"
	if err := c.StartTemplateBuild(ctx, templateID, buildID, apis.StartTemplateBuildV2JSONRequestBody{
		FromImage: &fromImage,
	}); err != nil {
		fmt.Printf("启动构建失败（可能已在构建中）: %v\n", err)
	} else {
		fmt.Println("构建已启动")
	}

	// 12. 等待构建完成（WaitForBuild）
	fmt.Println("\n=== 等待构建完成 ===")
	finalBuild, err := c.WaitForBuild(ctx, templateID, buildID, 3*time.Second)
	if err != nil {
		fmt.Printf("等待构建完成失败: %v\n", err)
	} else {
		fmt.Printf("构建已完成: %s (状态: %s)\n", finalBuild.BuildID, finalBuild.Status)
	}

	// 13. 删除模板（通过 defer 执行）
}
