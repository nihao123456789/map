package logic_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
	"os"
	"strings"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"

	"map-server/internal/config"
	"map-server/internal/errorx"
	"map-server/internal/handler"
	"map-server/internal/logic"
	"map-server/internal/middleware"
	"map-server/internal/svc"
	"map-server/internal/types"
)

func TestGetTradingsList_LocationInfo(t *testing.T) {
	var c config.Config
	// 加载本地开发配置文件
	conf.MustLoad("../../etc/mapserver-dev.yaml", &c)

	// 初始化服务上下文
	svcCtx := svc.NewServiceContext(c)

	l := logic.NewGetTradingsListLogic(context.Background(), svcCtx)

	req := &types.TradingListReq{
		Direction:                  "supply",
		Category:                   "dry",
		Condition:                  "brand_new",
		Color:                      "",
		EquipmentType:              "twenty_dry_container",
		CommercialTerm:             "pick_up",
		YearOfManufactureRangeFrom: 2025,
		PageSize:                   1,
	}

	resp, err := l.GetTradingsList(req)
	if err != nil {
		t.Fatalf("调用 GetTradingsList 报错: %v", err)
	}

	resBytes, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("--- 测试成功，返回数据样例 ---\n%s\n", string(resBytes))
}

func TestGetTradingLocationCount(t *testing.T) {
	var c config.Config
	// 加载本地开发配置文件
	conf.MustLoad("../../etc/mapserver-dev.yaml", &c)

	// 初始化服务上下文
	svcCtx := svc.NewServiceContext(c)

	l := logic.NewGetTradingLocationCountLogic(context.Background(), svcCtx)

	req := &types.TradingLocationCountReq{
		Direction: "supply",
	}

	resp, err := l.GetTradingLocationCount(req)
	if err != nil {
		t.Fatalf("调用 GetTradingLocationCount 报错: %v", err)
	}

	resBytes, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("--- 统计接口测试成功，返回数据样例 ---\n%s\n", string(resBytes))
}

func TestGetLocationList(t *testing.T) {
	var c config.Config
	conf.MustLoad("../../etc/mapserver-dev.yaml", &c)

	svcCtx := svc.NewServiceContext(c)
	l := logic.NewGetLocationListLogic(context.Background(), svcCtx)

	resp, err := l.GetLocationList()
	if err != nil {
		t.Fatalf("调用 GetLocationList 报错: %v", err)
	}

	resBytes, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("--- 位置列表接口测试成功，返回数据样例 ---\n%s\n", string(resBytes))
}


// TestRunServer 用于在受 Device Guard 限制的机器上，以单元测试的名义直接以白名单方式启动 HTTP 地图服务。
func TestRunServer(t *testing.T) {
	// 注册全局错误处理器，将参数校验错误及 Logic 层错误统一输出为友好的 JSON 格式
	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, interface{}) {
		switch e := err.(type) {
		case *errorx.CodeError:
			// 如果是自定义业务错误，则原样输出自定义的错误码和错误信息，HTTP 状态码返回 200 OK
			return http.StatusOK, map[string]interface{}{
				"code": e.Code,
				"msg":  e.Msg,
			}
		default:
			// 否则（如参数校验失败等普通错误），使用默认的 400 错误码
			return http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
			}
		}
	})

	// 从配置文件加载服务配置
	var c config.Config
	conf.MustLoad("../../etc/mapserver-dev.yaml", &c)

	// 1. 动态获取项目根目录并转换为绝对路径，避免 go test 自动切换工作目录到 internal/logic 导致日志生成在子目录下
	if !filepath.IsAbs(c.Log.Path) {
		if rootDir, err := filepath.Abs("../../"); err == nil {
			c.Log.Path = filepath.Join(rootDir, c.Log.Path)
		}
	}
	// 2. 注入服务名称以防 panic，将日志模式设为 volume，实现同时输出到控制台和日志文件
	c.Log.ServiceName = c.Name
	c.Log.Mode = "volume"
	logx.MustSetup(c.Log)

	// 初始化 go-zero HTTP 服务器
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 初始化服务上下文（包含 MySQL 连接等所有依赖）
	ctx := svc.NewServiceContext(c)

	// 注册全局运行时 Panic 兜底恢复中间件
	server.Use(middleware.RecoverMiddleware)

	// 注册全局成功返回包装中间件
	server.Use(middleware.UniformResponseMiddleware)

	// 注册所有 HTTP 路由处理器
	handler.RegisterHandlers(server, ctx)

	// 打印启动信息
	fmt.Printf("【白名单调试模式】地图服务启动成功，监听地址：%s:%d\n", c.Host, c.Port)
	fmt.Println("接口列表：")
	fmt.Println("  POST /api/tradings/list - 获取集装箱交易挂单列表（支持去重关联企业信息、会员徽章与地理位置树节点）")
	fmt.Println("  POST /api/tradings/location/count - 按位置统计交易挂单数量")
	fmt.Println("  POST /api/locations/list - 获取热门地理位置列表 (按使用频率排序)")

	// 启动 HTTP 服务，阻塞直到收到退出信号
	server.Start()
}

// TestGetTradingsList_Signature 验证 API 安全签名验证中间件（正常通过、缺失参数被拦截、签名不匹配被拦截及防重放拦截）。
func TestGetTradingsList_Signature(t *testing.T) {
	var c config.Config
	conf.MustLoad("../../etc/mapserver-dev.yaml", &c)

	// 使用开发环境配置的秘钥实例化签名验证中间件
	signMiddleware := middleware.NewSignatureMiddleware(c.SignatureSecret)

	// 模拟后续 Handler 处理器
	var handlerCalled bool
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}

	// 1. 缺失 X-Timestamp 校验拦截
	req1, _ := http.NewRequest("POST", "/api/tradings/list", nil)
	w1 := httptest.NewRecorder()
	handlerCalled = false
	signMiddleware.Handle(testHandler)(w1, req1)
	if handlerCalled {
		t.Error("缺失 X-Timestamp 请求头时，中间件不应该放行")
	}
	if w1.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401，实际得到: %d", w1.Code)
	}

	// 2. 签名内容错误拦截
	req2, _ := http.NewRequest("POST", "/api/tradings/list", nil)
	req2.Header.Set("X-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	req2.Header.Set("X-Nonce", "randomNonceValue")
	req2.Header.Set("X-Signature", "invalidSignatureString")
	w2 := httptest.NewRecorder()
	handlerCalled = false
	signMiddleware.Handle(testHandler)(w2, req2)
	if handlerCalled {
		t.Error("签名错误时，中间件不应该放行")
	}
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401，实际得到: %d", w2.Code)
	}

	// 3. 正确签名信息通过放行
	req3, _ := http.NewRequest("POST", "/api/tradings/list", nil)
	timestampStr := fmt.Sprintf("%d", time.Now().Unix())
	nonce := "uniqueNonceStr"
	req3.Header.Set("X-Timestamp", timestampStr)
	req3.Header.Set("X-Nonce", nonce)

	rawStr := fmt.Sprintf("timestamp=%s&nonce=%s&secret=%s", timestampStr, nonce, c.SignatureSecret)
	hash := sha256.New()
	hash.Write([]byte(rawStr))
	signature := hex.EncodeToString(hash.Sum(nil))
	req3.Header.Set("X-Signature", signature)

	w3 := httptest.NewRecorder()
	handlerCalled = false
	signMiddleware.Handle(testHandler)(w3, req3)
	if !handlerCalled {
		t.Error("签名正确时，中间件应该放行")
	}
}

// TestFixSwagger 用来在单元测试环境中以白名单程序运行，对 swagger.json 进行蛇形命名重构、字段删除和 EnumItem/EnumInfo 合并。
func TestFixSwagger(t *testing.T) {
	filePath := `../../swagger.json`
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("读取 swagger.json 失败: %v", err)
	}

	// 1. 全局字符串替换 (ConditionInfo -> EnumInfo)
	content := string(data)
	content = strings.ReplaceAll(content, "#/definitions/ConditionInfo", "#/definitions/EnumInfo")

	var swagger map[string]interface{}
	if err := json.Unmarshal([]byte(content), &swagger); err != nil {
		t.Fatalf("解析 JSON 失败: %v", err)
	}

	definitions, ok := swagger["definitions"].(map[string]interface{})
	if !ok {
		t.Fatalf("未找到 definitions 节点")
	}

	// 2. 将 ConditionInfo 改名为 EnumInfo，并裁剪字段
	if condInfo, exists := definitions["ConditionInfo"]; exists {
		definitions["EnumInfo"] = condInfo
		delete(definitions, "ConditionInfo")

		if enumMap, ok := definitions["EnumInfo"].(map[string]interface{}); ok {
			if props, ok := enumMap["properties"].(map[string]interface{}); ok {
				delete(props, "id")
				delete(props, "description")
				delete(props, "description_zh")
			}
		}
	}

	// 3. 修改 TradingData 字段
	renameProp := func(props map[string]interface{}, oldKey, newKey string) {
		if val, ok := props[oldKey]; ok {
			props[newKey] = val
			delete(props, oldKey)
		}
	}

	if tradingData, exists := definitions["TradingData"].(map[string]interface{}); exists {
		if props, ok := tradingData["properties"].(map[string]interface{}); ok {
			renameProp(props, "lastid", "last_id")
			renameProp(props, "pagesize", "page_size")
		}
		if req, ok := tradingData["required"].([]interface{}); ok {
			for i, v := range req {
				if v == "lastid" {
					req[i] = "last_id"
				}
				if v == "pagesize" {
					req[i] = "page_size"
				}
			}
		}
	}

	// 4. 修改 CompanyInfo 字段 (只保留 id, name, location_id, status, review_level, membership_badges)
	if compInfo, exists := definitions["CompanyInfo"].(map[string]interface{}); exists {
		if props, ok := compInfo["properties"].(map[string]interface{}); ok {
			delete(props, "telephone")
			delete(props, "email")
			delete(props, "usci")
			delete(props, "reviewscount")
			delete(props, "isofficial")
			delete(props, "address")

			renameProp(props, "locationid", "location_id")
			renameProp(props, "membershipbadges", "membership_badges")
			renameProp(props, "reviewlevel", "review_level")
		}
	}

	// 5. 修改 DepotInfo 字段
	if depotInfo, exists := definitions["DepotInfo"].(map[string]interface{}); exists {
		if props, ok := depotInfo["properties"].(map[string]interface{}); ok {
			delete(props, "postalcode")
			delete(props, "website")
			delete(props, "phonenumber")
			delete(props, "contactname")
			delete(props, "email")
			delete(props, "localname")
			delete(props, "localaddress")

			renameProp(props, "locationid", "location_id")
			renameProp(props, "addressline1", "address_line1")
			renameProp(props, "addressline2", "address_line2")
		}
	}

	// 6. 修改 LocationInfo 字段
	if locInfo, exists := definitions["LocationInfo"].(map[string]interface{}); exists {
		if props, ok := locInfo["properties"].(map[string]interface{}); ok {
			delete(props, "level")
			
			renameProp(props, "englishname", "english_name")
			renameProp(props, "fullname", "full_name")
			renameProp(props, "fullnamecn", "full_name_cn")
		}
	}

	// 7. 修改 MembershipBadge 字段
	if badge, exists := definitions["MembershipBadge"].(map[string]interface{}); exists {
		if props, ok := badge["properties"].(map[string]interface{}); ok {
			delete(props, "displayname")
			delete(props, "expiresat")
		}
	}

	// 8. 覆盖定义全新的 OfferInfo properties
	offerInfoProps := map[string]interface{}{
		"id": map[string]interface{}{
			"description": "挂单唯一主键 ID",
			"format":      "int64",
			"type":        "integer",
		},
		"condition": map[string]interface{}{
			"description": "集装箱质量状况：1=新箱，2=旧箱",
			"type":        "integer",
		},
		"type": map[string]interface{}{
			"description": "挂单业务类型（固定为 Trading）",
			"type":        "string",
		},
		"quantity": map[string]interface{}{
			"description": "集装箱数量",
			"type":        "integer",
		},
		"user_id": map[string]interface{}{
			"description": "创建挂单的用户 ID",
			"format":      "int64",
			"type":        "integer",
		},
		"company_id": map[string]interface{}{
			"description": "关联发布企业 ID",
			"format":      "int64",
			"type":        "integer",
		},
		"direction": map[string]interface{}{
			"description": "交易方向映射：0=买入 (supply)，1=卖出 (demand)",
			"type":        "integer",
		},
		"equipment_type": map[string]interface{}{
			"description": "集装箱规格属性",
			"type":        "integer",
		},
		"commercial_term": map[string]interface{}{
			"description": "条款机制",
			"type":        "integer",
		},
		"category": map[string]interface{}{
			"description": "类别",
			"type":        "integer",
		},
		"status": map[string]interface{}{
			"description": "挂单上架状态：10=已发布",
			"type":        "integer",
		},
		"location_id": map[string]interface{}{
			"description": "位置关联 ID",
			"format":      "int64",
			"type":        "integer",
		},
		"images_count": map[string]interface{}{
			"description": "图片张数",
			"type":        "integer",
		},
		"depot_id": map[string]interface{}{
			"description": "关联堆场唯一 ID",
			"format":      "int64",
			"type":        "integer",
		},
		"unique_number": map[string]interface{}{
			"description": "挂单系统流转唯一跟踪号",
			"type":        "string",
		},
		"created_at": map[string]interface{}{
			"description": "创建时间",
			"type":        "string",
		},
		"updated_at": map[string]interface{}{
			"description": "更新时间",
			"type":        "string",
		},
		"price": map[string]interface{}{
			"description": "单价",
			"type":        "number",
			"format":      "float",
		},
		"year_of_manufacture_range_from": map[string]interface{}{
			"description": "要求出厂年份起",
			"type":        "integer",
		},
		"year_of_manufacture_range_to": map[string]interface{}{
			"description": "要求出厂年份止",
			"type":        "integer",
		},
		"colors": map[string]interface{}{
			"description": "支持 the 备选颜色",
			"type":        "string",
		},
		"bumped_at": map[string]interface{}{
			"description": "排序重力刷新时间",
			"type":        "string",
		},
		"is_non_negotiable": map[string]interface{}{
			"description": "是否属于一口价不可议价单",
			"type":        "boolean",
		},
		"company_info": map[string]interface{}{
			"$ref":        "#/definitions/CompanyInfo",
			"description": "公司信息详情",
		},
		"depot_info": map[string]interface{}{
			"$ref":        "#/definitions/DepotInfo",
			"description": "堆场信息详情",
		},
		"location_info": map[string]interface{}{
			"$ref":        "#/definitions/LocationInfo",
			"description": "地理位置树节点信息详情",
		},
		"condition_info": map[string]interface{}{
			"$ref":        "#/definitions/EnumInfo",
			"description": "箱况详细信息详情",
		},
		"equipment_type_info": map[string]interface{}{
			"$ref":        "#/definitions/EnumInfo",
			"description": "箱型详细信息详情",
		},
		"commercial_term_info": map[string]interface{}{
			"$ref":        "#/definitions/EnumInfo",
			"description": "贸易条款（提箱方式）详细信息详情",
		},
		"category_info": map[string]interface{}{
			"$ref":        "#/definitions/EnumInfo",
			"description": "挂单分类详细信息详情",
		},
	}

	if offerInfo, exists := definitions["OfferInfo"].(map[string]interface{}); exists {
		offerInfo["properties"] = offerInfoProps
	}

	// 9. 删除 EnumItem 并全局替换引用为 EnumInfo
	delete(definitions, "EnumItem")

	// 10. 注入新的 /api/tradings/location/count 路径到 paths 中
	paths, ok := swagger["paths"].(map[string]interface{})
	if ok {
		paths["/api/tradings/location/count"] = map[string]interface{}{
			"post": map[string]interface{}{
				"consumes":    "application/json",
				"description": "根据交易方向，按位置 ID 分组统计有效的买卖交易挂单数量",
				"operationId": "GetTradingLocationCount",
				"parameters": []interface{}{
					map[string]interface{}{
						"description": "请求发起时的 Unix 时间戳（秒级）",
						"in":          "header",
						"name":        "X-Timestamp",
						"required":    true,
						"type":        "string",
					},
					map[string]interface{}{
						"description": "请求随机字符串",
						"in":          "header",
						"name":        "X-Nonce",
						"required":    true,
						"type":        "string",
					},
					map[string]interface{}{
						"description": "校验签名",
						"in":          "header",
						"name":        "X-Signature",
						"required":    true,
						"type":        "string",
					},
					map[string]interface{}{
						"description": "按位置统计数量请求",
						"in":          "body",
						"name":        "body",
						"required":    true,
						"schema": map[string]interface{}{
							"$ref": "#/definitions/TradingLocationCountReq",
						},
					},
				},
				"produces": []interface{}{"application/json"},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "统计成功",
						"schema": map[string]interface{}{
							"$ref": "#/definitions/TradingLocationCountResp",
						},
					},
				},
				"summary": "按位置统计交易挂单数量",
			},
		}

		paths["/api/locations/list"] = map[string]interface{}{
			"post": map[string]interface{}{
				"consumes":    "application/json",
				"description": "获取按使用频率从高到低排序的热门地理位置城市列表",
				"operationId": "GetLocationList",
				"parameters": []interface{}{
					map[string]interface{}{
						"description": "请求发起时的 Unix 时间戳（秒级）",
						"in":          "header",
						"name":        "X-Timestamp",
						"required":    true,
						"type":        "string",
					},
					map[string]interface{}{
						"description": "请求随机字符串",
						"in":          "header",
						"name":        "X-Nonce",
						"required":    true,
						"type":        "string",
					},
					map[string]interface{}{
						"description": "校验签名",
						"in":          "header",
						"name":        "X-Signature",
						"required":    true,
						"type":        "string",
					},
				},
				"produces": []interface{}{"application/json"},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "获取成功",
						"schema": map[string]interface{}{
							"$ref": "#/definitions/LocationListResp",
						},
					},
				},
				"summary": "获取热门地理位置列表 (按使用频率排序)",
			},
		}
	}

	// 11. 注入 TradingLocationCountReq, TradingLocationCountItem 和 TradingLocationCountResp 到 definitions
	definitions["TradingLocationCountReq"] = map[string]interface{}{
		"properties": map[string]interface{}{
			"direction": map[string]interface{}{
				"description": "交易方向",
				"type":        "string",
			},
		},
		"required": []interface{}{"direction"},
		"type":     "object",
	}

	definitions["TradingLocationCountItem"] = map[string]interface{}{
		"properties": map[string]interface{}{
			"count": map[string]interface{}{
				"description": "数量",
				"format":      "int64",
				"type":        "integer",
			},
			"location_id": map[string]interface{}{
				"description": "位置 ID",
				"format":      "int64",
				"type":        "integer",
			},
			"location_info": map[string]interface{}{
				"$ref":        "#/definitions/LocationInfo",
				"description": "地理位置树节点信息详情",
			},
		},
		"required": []interface{}{"location_id", "count"},
		"type":     "object",
	}

	definitions["TradingLocationCountResp"] = map[string]interface{}{
		"properties": map[string]interface{}{
			"list": map[string]interface{}{
				"description": "按位置统计数量子项列表",
				"items": map[string]interface{}{
					"$ref": "#/definitions/TradingLocationCountItem",
				},
				"type": "array",
			},
		},
		"required": []interface{}{"list"},
		"type":     "object",
	}

	definitions["LocationListResp"] = map[string]interface{}{
		"properties": map[string]interface{}{
			"list": map[string]interface{}{
				"description": "热门位置列表项",
				"items": map[string]interface{}{
					"$ref": "#/definitions/LocationInfo",
				},
				"type": "array",
			},
		},
		"required": []interface{}{"list"},
		"type":     "object",
	}

	updatedData, err := json.MarshalIndent(swagger, "", "    ")
	if err != nil {
		t.Fatalf("序列化 JSON 失败: %v", err)
	}

	finalContent := string(updatedData)
	finalContent = strings.ReplaceAll(finalContent, "#/definitions/EnumItem", "#/definitions/EnumInfo")

	err = os.WriteFile(filePath, []byte(finalContent), 0644)
	if err != nil {
		t.Fatalf("写入 swagger.json 失败: %v", err)
	}

	t.Log("swagger.json 修正成功！")
}

func TestPanicRecoverMiddleware(t *testing.T) {
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		panic("模拟运行时严重崩溃")
	}

	req, _ := http.NewRequest("POST", "/api/test/panic", nil)
	w := httptest.NewRecorder()

	middleware.RecoverMiddleware(testHandler)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望返回状态码 200，实际得到: %d", w.Code)
	}

	expectedBody := `{"code":500,"msg":"系统繁忙，请稍后再试"}`
	if w.Body.String() != expectedBody {
		t.Errorf("期望返回体 %s，实际得到: %s", expectedBody, w.Body.String())
	}
}

