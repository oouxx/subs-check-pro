package assets

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

// TestDecompressSubStoreBackend 测试解压 sub-store.bundle.js.zst 文件
func TestDecompressSubStoreBackend(t *testing.T) {
	// 1. 验证 EmbeddedSubStoreBackend 变量不为空
	if len(EmbeddedSubStoreBackend) == 0 {
		t.Fatal("EmbeddedSubStoreBackend is empty")
	}
	t.Logf("EmbeddedSubStoreBackend size: %d bytes", len(EmbeddedSubStoreBackend))

	// 2. 创建临时目录用于解压文件
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "sub-store.bundle.js")
	t.Logf("Output path: %s", outputPath)

	// 3. 创建 zstd 解码器
	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatalf("Failed to create zstd decoder: %v", err)
	}
	defer zstdDecoder.Close()

	// 4. 解压文件
	outputFile, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}
	defer outputFile.Close()

	zstdDecoder.Reset(bytes.NewReader(EmbeddedSubStoreBackend))
	decompressedSize, err := io.Copy(outputFile, zstdDecoder)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	t.Logf("Decompressed size: %d bytes", decompressedSize)

	// 5. 验证解压后的文件
	// 重新打开文件进行验证
	outputFile.Seek(0, 0)
	fileInfo, err := outputFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Fatal("Decompressed file is empty")
	}

	if fileInfo.Size() < 1000 {
		t.Logf("Warning: Decompressed file is very small (%d bytes), might be incomplete", fileInfo.Size())
	}

	// 6. 检查文件内容是否是 JavaScript
	// 读取前 1KB 来检查文件类型
	buffer := make([]byte, 1024)
	n, err := outputFile.Read(buffer)
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read output file: %v", err)
	}

	content := string(buffer[:n])

	// 检查常见的 JavaScript 模式
	jsPatterns := []string{
		"function",
		"var ",
		"const ",
		"let ",
		"export ",
		"import ",
		"//",
		"/*",
		"module.exports",
		"require(",
	}

	foundPatterns := []string{}
	for _, pattern := range jsPatterns {
		if strings.Contains(content, pattern) {
			foundPatterns = append(foundPatterns, pattern)
		}
	}

	if len(foundPatterns) == 0 {
		t.Log("Warning: No common JavaScript patterns found in decompressed content")
		t.Logf("First 500 chars of content: %s", content[:min(500, len(content))])
	} else {
		t.Logf("Found JavaScript patterns: %v", foundPatterns)
	}

	// 7. 验证压缩比
	compressionRatio := float64(len(EmbeddedSubStoreBackend)) / float64(decompressedSize)
	t.Logf("Compression ratio: %.2f (compressed/original)", compressionRatio)

	// 8. 验证文件可以被正常读取为文本
	// 尝试读取整个文件（如果不太大）
	if decompressedSize < 10*1024*1024 { // 小于 10MB
		outputFile.Seek(0, 0)
		fullContent, err := io.ReadAll(outputFile)
		if err != nil {
			t.Logf("Note: Could not read full file: %v", err)
		} else {
			// 检查文件是否包含 null 字节（二进制文件的特征）
			if bytes.Contains(fullContent, []byte{0}) {
				t.Log("File contains null bytes, might be binary or minified JavaScript")
			} else {
				t.Log("File appears to be text-based (no null bytes)")
			}

			// 检查文件行数
			lineCount := strings.Count(string(fullContent), "\n")
			t.Logf("File has approximately %d lines", lineCount)
		}
	}

	t.Log("Test passed: sub-store.bundle.js.zst successfully decompressed")
}

// TestDecompressAndValidate 更详细的验证测试
func TestDecompressAndValidate(t *testing.T) {
	// 创建多个解码器实例进行测试
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "basic decompression",
			testFunc: func(t *testing.T) {
				zstdDecoder, err := zstd.NewReader(nil)
				if err != nil {
					t.Fatal(err)
				}
				defer zstdDecoder.Close()

				zstdDecoder.Reset(bytes.NewReader(EmbeddedSubStoreBackend))
				decompressed, err := io.ReadAll(zstdDecoder)
				if err != nil {
					t.Fatal(err)
				}

				if len(decompressed) == 0 {
					t.Fatal("Decompressed data is empty")
				}

				// 验证不是全零或全相同字节
				allSame := true
				firstByte := decompressed[0]
				for _, b := range decompressed[1:] {
					if b != firstByte {
						allSame = false
						break
					}
				}
				if allSame {
					t.Fatal("Decompressed data appears to be all the same byte")
				}
			},
		},
		{
			name: "multiple decompression passes",
			testFunc: func(t *testing.T) {
				// 多次解压确保一致性
				var previousSize int64
				for i := 0; i < 3; i++ {
					zstdDecoder, err := zstd.NewReader(nil)
					if err != nil {
						t.Fatal(err)
					}

					zstdDecoder.Reset(bytes.NewReader(EmbeddedSubStoreBackend))
					decompressed, err := io.ReadAll(zstdDecoder)
					zstdDecoder.Close()

					if err != nil {
						t.Fatal(err)
					}

					currentSize := int64(len(decompressed))
					if i > 0 && currentSize != previousSize {
						t.Fatalf("Decompression size mismatch: pass %d = %d, pass %d = %d",
							i-1, previousSize, i, currentSize)
					}
					previousSize = currentSize
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

// TestEmbeddedFileExists 测试嵌入文件是否存在且有效
func TestEmbeddedFileExists(t *testing.T) {
	// 检查文件是否以 zstd 魔数开头
	// zstd 文件通常以 0xFD2FB528 开头（小端序）
	if len(EmbeddedSubStoreBackend) < 4 {
		t.Fatal("Embedded file too small to contain zstd magic number")
	}

	// zstd 魔数：0xFD2FB528
	zstdMagic := []byte{0x28, 0xB5, 0x2F, 0xFD}
	if bytes.Equal(EmbeddedSubStoreBackend[:4], zstdMagic) {
		t.Log("File has valid zstd magic number")
	} else {
		// 可能是没有魔数的帧
		t.Log("File does not have standard zstd magic number (might be skippable frame)")
	}

	// 尝试快速解码验证
	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer zstdDecoder.Close()

	// 只解码一小部分来验证格式
	zstdDecoder.Reset(bytes.NewReader(EmbeddedSubStoreBackend))
	smallBuffer := make([]byte, 1024)
	_, err = zstdDecoder.Read(smallBuffer)
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to decode zstd format: %v", err)
	}

	t.Log("Embedded file appears to be valid zstd compressed data")
}

// BenchmarkDecompress 性能测试
func BenchmarkDecompress(b *testing.B) {
	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		b.Fatal(err)
	}
	defer zstdDecoder.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		zstdDecoder.Reset(bytes.NewReader(EmbeddedSubStoreBackend))
		_, err := io.Copy(io.Discard, zstdDecoder)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestDecompressACL4SSR 测试解压 ACL4SSR_Online_Full.yaml.zst 文件
func TestDecompressACL4SSR(t *testing.T) {
	// 1. 验证 EmbeddedOverrideYaml 变量不为空
	if len(EmbeddedOverrideYaml) == 0 {
		t.Fatal("EmbeddedOverrideYaml is empty")
	}
	t.Logf("EmbeddedOverrideYaml size: %d bytes", len(EmbeddedOverrideYaml))

	// 2. 创建临时目录用于解压文件
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "ACL4SSR_Online_Full.yaml")
	t.Logf("Output path: %s", outputPath)

	// 3. 创建 zstd 解码器
	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatalf("Failed to create zstd decoder: %v", err)
	}
	defer zstdDecoder.Close()

	// 4. 解压文件
	outputFile, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}
	defer outputFile.Close()

	zstdDecoder.Reset(bytes.NewReader(EmbeddedOverrideYaml))
	decompressedSize, err := io.Copy(outputFile, zstdDecoder)
	if err != nil {
		t.Fatalf("Failed to decompress ACL4SSR file: %v", err)
	}

	t.Logf("Decompressed size: %d bytes", decompressedSize)

	// 5. 验证解压后的文件
	// 重新打开文件进行验证
	outputFile.Seek(0, 0)
	fileInfo, err := outputFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Fatal("Decompressed file is empty")
	}

	if fileInfo.Size() < 100 {
		t.Logf("Warning: Decompressed file is very small (%d bytes), might be incomplete", fileInfo.Size())
	}

	// 6. 检查文件内容是否是 YAML 格式
	// 读取前 2KB 来检查文件类型
	buffer := make([]byte, 2048)
	n, err := outputFile.Read(buffer)
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read output file: %v", err)
	}

	content := string(buffer[:n])

	// 检查常见的 YAML 模式
	yamlPatterns := []string{
		"#",
		"---",
		"rules:",
		"payload:",
		"proxies:",
		"proxy-groups:",
		"rule-providers:",
		"  - ",
		"  GEOIP",
		"  DOMAIN",
		"  IP-CIDR",
		"  FINAL",
	}

	foundPatterns := []string{}
	for _, pattern := range yamlPatterns {
		if strings.Contains(content, pattern) {
			foundPatterns = append(foundPatterns, pattern)
		}
	}

	if len(foundPatterns) == 0 {
		t.Log("Warning: No common YAML patterns found in decompressed content")
		t.Logf("First 500 chars of content: %s", content[:min(500, len(content))])
	} else {
		t.Logf("Found YAML patterns: %v", foundPatterns)
	}

	// 7. 验证压缩比
	compressionRatio := float64(len(EmbeddedOverrideYaml)) / float64(decompressedSize)
	t.Logf("Compression ratio: %.2f (compressed/original)", compressionRatio)

	// 8. 验证文件是有效的 YAML 文本
	outputFile.Seek(0, 0)
	fullContent, err := io.ReadAll(outputFile)
	if err != nil {
		t.Logf("Note: Could not read full file: %v", err)
	} else {
		// 检查文件是否包含 null 字节（二进制文件的特征）
		if bytes.Contains(fullContent, []byte{0}) {
			t.Log("File contains null bytes, might not be pure text")
		} else {
			t.Log("File appears to be text-based (no null bytes)")
		}

		// 检查文件行数
		lineCount := strings.Count(string(fullContent), "\n")
		t.Logf("File has approximately %d lines", lineCount)

		// 检查是否是有效的 YAML（基本检查）
		if strings.Contains(string(fullContent), "rules:") ||
			strings.Contains(string(fullContent), "payload:") ||
			strings.Contains(string(fullContent), "#") {
			t.Log("File appears to be a valid ACL4SSR YAML configuration")
		}

		// 检查文件大小是否合理（ACL4SSR 规则文件通常较大）
		if decompressedSize > 1024*1024 { // 大于 1MB
			t.Log("File is large, typical for ACL4SSR rule sets")
		}
	}

	t.Log("Test passed: ACL4SSR_Online_Full.yaml.zst successfully decompressed")
}

// TestDecompressAllEmbeddedFiles 测试解压所有嵌入文件
func TestDecompressAllEmbeddedFiles(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		outputName  string
		description string
	}{
		{
			name:        "SubStoreBackend",
			data:        EmbeddedSubStoreBackend,
			outputName:  "sub-store.bundle.js",
			description: "Sub-Store 后端 JavaScript 文件",
		},
		{
			name:        "SubStoreFrontend",
			data:        EmbeddedSubStoreFrotend,
			outputName:  "sub-store.frontend.tar",
			description: "Sub-Store 前端文件（tar 格式）",
		},
		{
			name:        "ACL4SSR",
			data:        EmbeddedOverrideYaml,
			outputName:  "ACL4SSR_Online_Full.yaml",
			description: "ACL4SSR 覆写规则文件",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证数据不为空
			if len(tt.data) == 0 {
				t.Fatalf("%s is empty", tt.description)
			}
			t.Logf("%s size: %d bytes", tt.description, len(tt.data))

			// 创建临时目录
			tempDir := t.TempDir()
			outputPath := filepath.Join(tempDir, tt.outputName)

			// 创建 zstd 解码器
			zstdDecoder, err := zstd.NewReader(nil)
			if err != nil {
				t.Fatalf("Failed to create zstd decoder: %v", err)
			}
			defer zstdDecoder.Close()

			// 解压文件
			outputFile, err := os.Create(outputPath)
			if err != nil {
				t.Fatalf("Failed to create output file: %v", err)
			}
			defer outputFile.Close()

			zstdDecoder.Reset(bytes.NewReader(tt.data))
			decompressedSize, err := io.Copy(outputFile, zstdDecoder)
			if err != nil {
				t.Fatalf("Failed to decompress %s: %v", tt.description, err)
			}

			t.Logf("%s decompressed size: %d bytes", tt.description, decompressedSize)

			// 验证解压后的文件不为空
			if decompressedSize == 0 {
				t.Fatalf("Decompressed %s is empty", tt.description)
			}

			// 验证压缩比
			compressionRatio := float64(len(tt.data)) / float64(decompressedSize)
			t.Logf("%s compression ratio: %.2f", tt.description, compressionRatio)
		})
	}
}

// TestACL4SSRContentValidation 验证 ACL4SSR 文件内容
func TestACL4SSRContentValidation(t *testing.T) {
	// 1. 解压文件到内存
	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer zstdDecoder.Close()

	zstdDecoder.Reset(bytes.NewReader(EmbeddedOverrideYaml))
	decompressedData, err := io.ReadAll(zstdDecoder)
	if err != nil {
		t.Fatalf("Failed to decompress ACL4SSR: %v", err)
	}

	if len(decompressedData) == 0 {
		t.Fatal("Decompressed data is empty")
	}

	content := string(decompressedData)

	// 2. 验证基本的 YAML 结构
	requiredSections := []string{
		"rules:",
		"payload:",
	}

	foundSections := []string{}
	for _, section := range requiredSections {
		if strings.Contains(content, section) {
			foundSections = append(foundSections, section)
		}
	}

	if len(foundSections) == 0 {
		t.Log("Warning: No standard ACL4SSR sections found")
	} else {
		t.Logf("Found ACL4SSR sections: %v", foundSections)
	}

	// 3. 验证规则数量（通过行数估算）
	lines := strings.Split(content, "\n")
	ruleLines := 0
	commentLines := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			commentLines++
		} else if strings.HasPrefix(trimmed, "- ") ||
			strings.HasPrefix(trimmed, "  - ") ||
			strings.Contains(trimmed, ",") {
			ruleLines++
		}
	}

	t.Logf("Total lines: %d", len(lines))
	t.Logf("Comment lines: %d", commentLines)
	t.Logf("Estimated rule lines: %d", ruleLines)

	// 4. 验证文件以 YAML 文档分隔符或注释开头
	if !strings.HasPrefix(strings.TrimSpace(content), "#") &&
		!strings.HasPrefix(strings.TrimSpace(content), "---") {
		t.Log("Warning: File does not start with comment or YAML document separator")
	}

	// 5. 验证包含常见的规则类型
	commonRuleTypes := []string{
		"GEOIP",
		"DOMAIN",
		"DOMAIN-SUFFIX",
		"DOMAIN-KEYWORD",
		"IP-CIDR",
		"IP-CIDR6",
		"SRC-IP-CIDR",
		"PROCESS-NAME",
		"FINAL",
	}

	foundRuleTypes := []string{}
	for _, ruleType := range commonRuleTypes {
		if strings.Contains(content, ruleType) {
			foundRuleTypes = append(foundRuleTypes, ruleType)
		}
	}

	if len(foundRuleTypes) > 0 {
		t.Logf("Found rule types: %v", foundRuleTypes)
	} else {
		t.Log("Warning: No common rule types found")
	}

	// 6. 验证文件大小合理
	fileSize := len(decompressedData)
	if fileSize < 1024 { // 小于 1KB
		t.Log("Warning: ACL4SSR file is unusually small")
	} else if fileSize > 10*1024*1024 { // 大于 10MB
		t.Log("Warning: ACL4SSR file is unusually large")
	} else {
		t.Logf("ACL4SSR file size is reasonable: %d bytes", fileSize)
	}
}

// BenchmarkDecompressACL4SSR ACL4SSR 解压性能测试
func BenchmarkDecompressACL4SSR(b *testing.B) {
	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		b.Fatal(err)
	}
	defer zstdDecoder.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		zstdDecoder.Reset(bytes.NewReader(EmbeddedOverrideYaml))
		_, err := io.Copy(io.Discard, zstdDecoder)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestDecompressACL4SSRToCurrentDir 测试解压 ACL4SSR_Online_Full.yaml.zst 文件到当前目录
func TestDecompressACL4SSRToCurrentDir(t *testing.T) {
	// 1. 验证 EmbeddedOverrideYaml 变量不为空
	if len(EmbeddedOverrideYaml) == 0 {
		t.Fatal("EmbeddedOverrideYaml is empty")
	}
	t.Logf("EmbeddedOverrideYaml size: %d bytes", len(EmbeddedOverrideYaml))

	// 2. 使用当前目录而不是临时目录
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// 创建一个专门的测试输出目录
	outputDir := filepath.Join(currentDir, "test_output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}
	defer func() {
		// 测试结束后不删除文件，方便查看
		t.Logf("Output files preserved in: %s", outputDir)
	}()

	outputPath := filepath.Join(outputDir, "ACL4SSR_Online_Full.yaml")
	t.Logf("Output path: %s", outputPath)

	// 3. 创建 zstd 解码器
	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatalf("Failed to create zstd decoder: %v", err)
	}
	defer zstdDecoder.Close()

	// 4. 解压文件
	outputFile, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}
	defer outputFile.Close()

	zstdDecoder.Reset(bytes.NewReader(EmbeddedOverrideYaml))
	decompressedSize, err := io.Copy(outputFile, zstdDecoder)
	if err != nil {
		t.Fatalf("Failed to decompress ACL4SSR file: %v", err)
	}

	t.Logf("Decompressed size: %d bytes", decompressedSize)

	// 5. 验证解压后的文件
	// 重新打开文件进行验证
	outputFile.Seek(0, 0)
	fileInfo, err := outputFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Fatal("Decompressed file is empty")
	}

	t.Logf("File saved to: %s", outputPath)
	t.Logf("File size: %d bytes", fileInfo.Size())

	// 6. 显示文件前几行内容
	outputFile.Seek(0, 0)
	firstLines := make([]byte, 2000) // 读取前2000字节
	n, err := outputFile.Read(firstLines)
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read output file: %v", err)
	}

	content := string(firstLines[:n])
	lines := strings.Split(content, "\n")

	// 显示前20行
	t.Log("First 20 lines of decompressed file:")
	for i := 0; i < min(20, len(lines)); i++ {
		t.Logf("  Line %3d: %s", i+1, lines[i])
	}

	// 7. 检查文件内容
	if strings.Contains(content, "#") {
		t.Log("File contains comments (good sign for YAML)")
	}
	if strings.Contains(content, "rules:") {
		t.Log("File contains 'rules:' section")
	}
	if strings.Contains(content, "payload:") {
		t.Log("File contains 'payload:' section")
	}

	// 8. 创建文件信息摘要
	summaryPath := filepath.Join(outputDir, "file_info.txt")
	summaryFile, err := os.Create(summaryPath)
	if err != nil {
		t.Logf("Failed to create summary file: %v", err)
	} else {
		defer summaryFile.Close()

		summary := fmt.Sprintf(`ACL4SSR File Information:
==========================
Original compressed size: %d bytes
Decompressed size: %d bytes
Compression ratio: %.2f
Output path: %s
File exists: %v
File size: %d bytes

First 10 lines:
`,
			len(EmbeddedOverrideYaml),
			decompressedSize,
			float64(len(EmbeddedOverrideYaml))/float64(decompressedSize),
			outputPath,
			fileInfo != nil,
			fileInfo.Size(),
		)

		summaryFile.WriteString(summary)
		for i := 0; i < min(10, len(lines)); i++ {
			summaryFile.WriteString(fmt.Sprintf("%3d: %s\n", i+1, lines[i]))
		}

		t.Logf("File information saved to: %s", summaryPath)
	}

	t.Log("Test passed: ACL4SSR_Online_Full.yaml.zst successfully decompressed to current directory")
}
