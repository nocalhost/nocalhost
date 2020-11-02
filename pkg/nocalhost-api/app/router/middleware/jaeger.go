/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package middleware

//
//import (
//	"github.com/gin-gonic/gin"
//	"github.com/opentracing/opentracing-go"
//	"github.com/opentracing/opentracing-go/ext"
//	"github.com/spf13/viper"
//	"go-gin-api/app/config"
//	"go-gin-api/app/util/jaeger_trace"
//	"io"
//)
//
//var Tracer opentracing.Tracer
//var Closer io.Closer
//var Error error
//
//var ParentSpan opentracing.Span
//
//func SetUp() gin.HandlerFunc {
//
//	return func(c *gin.Context) {
//		if viper.GetInt("jaeger_open") == 1 {
//			Tracer, Closer, Error = jaeger_trace.NewJaegerTracer(config.AppName, config.JaegerHostPort)
//			defer Closer.Close()
//
//			spCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders,
//			opentracing.HTTPHeadersCarrier(c.Request.Header))
//			if err != nil {
//				ParentSpan = Tracer.StartSpan(c.Request.URL.Path)
//				defer ParentSpan.Finish()
//			} else {
//				ParentSpan = opentracing.StartSpan(
//					c.Request.URL.Path,
//					opentracing.ChildOf(spCtx),
//					opentracing.Tag{Key: string(ext.Component), Value: "HTTP"},
//					ext.SpanKindRPCServer,
//				)
//				defer ParentSpan.Finish()
//			}
//		}
//		c.Next()
//	}
//}
