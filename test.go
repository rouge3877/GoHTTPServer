package main

// import (
// 	"fmt"
// )

// // ---------------------------
// // 接口定义
// // ---------------------------
// type Processor interface {
// 	DoSomething()
// }

// // ---------------------------
// // 基类定义
// // ---------------------------
// type BaseHandler struct {
// 	Processor Processor
// }

// func (b *BaseHandler) Handle() {
// 	fmt.Println("BaseHandler.Handle called")
// 	b.Processor.DoSomething() // 多态调用
// }

// func (b *BaseHandler) DoSomething() {
// 	fmt.Println("BaseHandler.DoSomething called")
// }

// // ---------------------------
// // 子类定义
// // ---------------------------
// type MyHandler struct {
// 	*BaseHandler
// 	Name string
// }

// func (m *MyHandler) DoSomething() {
// 	fmt.Printf("MyHandler.DoSomething called! Name = %s\n", m.Name)
// }

// // ---------------------------
// // 构造函数
// // ---------------------------
// func NewMyHandler(name string) *MyHandler {
// 	handler := &MyHandler{Name: name}
// 	base := &BaseHandler{Processor: handler}
// 	handler.BaseHandler = base
// 	return handler
// }

// type YourHandler struct {
// 	*BaseHandler
// 	Name string
// }

// func (m *YourHandler) DoSomething() {
// 	fmt.Printf("YourHandler.DoSomething called! Name = %s\n", m.Name)
// }

// // ---------------------------
// // 构造函数
// // ---------------------------
// func NewYourHandler(name string) *YourHandler {
// 	handler := &YourHandler{Name: name}
// 	base := &BaseHandler{Processor: handler}
// 	handler.BaseHandler = base
// 	return handler
// }

// // ---------------------------
// // 测试入口
// // ---------------------------
// func main() {
// 	handler := NewMyHandler("Alice")
// 	shandler := NewYourHandler("Bob")
// 	handler.Handle()
// 	shandler.Handle()

// 	fmt.Println("------")

// 	// 直接用基类调用时的区别（不会多态）
// 	base := &BaseHandler{Processor: handler}
// 	base.Handle()
// }
