/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"github.com/derailed/tview"
)

type EnhancedTreeNode struct {
	*tview.TreeNode
	Parent *EnhancedTreeNode
}

func (e *EnhancedTreeNode) AddChild(node *EnhancedTreeNode) *EnhancedTreeNode {
	e.TreeNode.AddChild(node.TreeNode)
	node.Parent = e
	return e
}
