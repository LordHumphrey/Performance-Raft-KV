// Package etcdapi provides etcd v3 API compatibility for the distributed key-value store.
package etcdapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/otoolep/hraftd/store"
	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// Service provides etcd v3 API service.
type Service struct {
	addr       string
	ln         net.Listener
	store      *store.Store
	srv        *grpc.Server
	httpServer *http.Server
}

// New returns an uninitialized etcd API service.
func New(addr string, store *store.Store) *Service {
	return &Service{
		addr:  addr,
		store: store,
	}
}

// Start starts the service.
func (s *Service) Start() error {
	// 创建一个 gRPC 服务器
	s.srv = grpc.NewServer()

	// 注册 KV 服务
	pb.RegisterKVServer(s.srv, s)

	// 启用 gRPC 反射服务，这对于调试和一些客户端很有用
	reflection.Register(s.srv)

	// 启动 gRPC 监听
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.ln = ln

	// 启动 gRPC 服务器
	go func() {
		if err := s.srv.Serve(ln); err != nil {
			log.Fatalf("etcd API gRPC serve: %s", err)
		}
	}()

	// 提取主机和端口
	host := s.addr
	portStr := ""
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		host = parts[0]
		portStr = parts[1]
	}

	// 创建一个 HTTP 服务器，将请求转发到 gRPC 服务器
	// 这是为了支持 go-ycsb 等使用 HTTP 协议的客户端
	// 使用相同的端口，但是 HTTP 服务器使用 gRPC 端口 + 10000
	port, _ := strconv.Atoi(portStr)
	httpPort := port + 10000
	httpAddr := fmt.Sprintf("%s:%d", host, httpPort)

	// 创建一个专用的 ServeMux，而不是使用全局的 DefaultServeMux
	mux := http.NewServeMux()

	// 注册 HTTP 处理程序
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 设置 CORS 头，允许跨域请求
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// 处理 OPTIONS 请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 创建到 gRPC 服务器的连接，添加超时控制
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, s.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			http.Error(w, "无法连接到 gRPC 服务器: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		// 创建 KV 客户端
		client := pb.NewKVClient(conn)

		// 根据请求路径和方法调用相应的 gRPC 方法
		switch {
		case r.URL.Path == "/v3/kv/put" && r.Method == "POST":
			// 处理 PUT 请求
			var req pb.PutRequest
			if err := decodeJSONRequest(r, &req); err != nil {
				http.Error(w, "无法解析请求: "+err.Error(), http.StatusBadRequest)
				return
			}

			// 使用带超时的上下文
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resp, err := client.Put(ctx, &req)
			if err != nil {
				http.Error(w, "Put 操作失败: "+err.Error(), http.StatusInternalServerError)
				return
			}

			encodeJSONResponse(w, resp)

		case r.URL.Path == "/v3/kv/range" && r.Method == "POST":
			// 处理 RANGE 请求
			var req pb.RangeRequest
			if err := decodeJSONRequest(r, &req); err != nil {
				http.Error(w, "无法解析请求: "+err.Error(), http.StatusBadRequest)
				return
			}

			// 使用带超时的上下文
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resp, err := client.Range(ctx, &req)
			if err != nil {
				http.Error(w, "Range 操作失败: "+err.Error(), http.StatusInternalServerError)
				return
			}

			encodeJSONResponse(w, resp)

		case r.URL.Path == "/v3/kv/deleterange" && r.Method == "POST":
			// 处理 DELETE 请求
			var req pb.DeleteRangeRequest
			if err := decodeJSONRequest(r, &req); err != nil {
				http.Error(w, "无法解析请求: "+err.Error(), http.StatusBadRequest)
				return
			}

			// 使用带超时的上下文
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resp, err := client.DeleteRange(ctx, &req)
			if err != nil {
				http.Error(w, "DeleteRange 操作失败: "+err.Error(), http.StatusInternalServerError)
				return
			}

			encodeJSONResponse(w, resp)

		default:
			// 对于其他请求，返回 404
			http.NotFound(w, r)
		}
	})

	// 启动 HTTP 服务器，使用我们创建的 ServeMux
	s.httpServer = &http.Server{
		Addr:         httpAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("etcd API HTTP serve: %s", err)
		}
	}()

	log.Printf("etcd API service listening on gRPC: %s, HTTP: %s", s.addr, httpAddr)

	return nil
}

// 从 HTTP 请求中解析 JSON
func decodeJSONRequest(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// 将响应编码为 JSON 并写入 HTTP 响应
func encodeJSONResponse(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "无法编码响应: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// Close closes the service.
func (s *Service) Close() {
	// 创建一个带超时的上下文用于关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if s.srv != nil {
		s.srv.GracefulStop()
	}
	if s.ln != nil {
		s.ln.Close()
	}
	if s.httpServer != nil {
		s.httpServer.Shutdown(ctx)
	}
}

// Addr returns the address on which the Service is listening
func (s *Service) Addr() net.Addr {
	return s.ln.Addr()
}
