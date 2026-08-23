# A Tour of Go 匿名翻译质量评审

这是匿名翻译质量评审。不得猜测 Candidate A/B/C 的来源；不得搜索、查阅或模仿任何现有 A Tour of Go 中文译文；不得融合三个 candidate 后给出新译文；只评价现有 Candidate A/B/C。

必须仅根据每页提供的英文原文、zh-CN glossary 和三份现有最终译文进行评分与排名。五个 TranslationUnit 必须分别独立评价；不得因为某个候选在上一页表现较好，就给下一页先验优势。

## 评分规则（总分 100）

- 技术准确性（30）：检查 Go 技术含义；类型、接口、并发、泛型、算法等概念；技术关系是否准确；是否出现会误导初学者的技术错误。
- 忠实原文（20）：检查是否遗漏信息、擅自增加解释；因果、范围、语气是否与原文一致；链接显示文字的含义是否准确。
- 中文自然度（20）：检查是否自然、是否有明显翻译腔、句式是否符合正式中文技术教程。
- 教学表达（15）：检查是否适合 Go 初学者、是否清晰、上下文衔接是否自然、是否准确传达原文教学意图。
- 术语一致性（10）：结合下方 glossary 检查。
- 可读性（5）：检查句子长度、标点、阅读节奏与整体易读程度。

## 排名规则

对每一个 TranslationUnit 分别给 Candidate A/B/C 打分，然后给出严格的第 1、2、3 名，原则上不得并列。如果两份非常接近，也必须选择略优的一份，并简短解释真正决定排名的细节。

## 错误严重度

- critical：存在明显技术错误，可能误导学习者。
- major：重要语义、忠实度或教学表达问题，明显影响质量。
- minor：措辞、自然度、局部表达等轻微问题。

不要把纯风格偏好夸大成技术错误。

## 最终输出要求

最终只输出一个 JSON 对象，不要输出 JSON 之外的解释性文字。必须包含下方全部 5 个 unit_id。每个分项不得超过对应满分，且 total 必须严格等于六项之和。结构如下：

```json
{
  "pages": [
    {
      "unit_id": "methods/24",
      "candidates": {
        "A": {
          "technical_accuracy": 0,
          "fidelity": 0,
          "chinese_naturalness": 0,
          "teaching_expression": 0,
          "terminology_consistency": 0,
          "readability": 0,
          "total": 0,
          "critical": [],
          "major": [],
          "minor": []
        },
        "B": {},
        "C": {}
      },
      "ranking": ["A", "B", "C"],
      "ranking_reason": "简要说明真正决定排名的差异"
    }
  ],
  "overall_observations": "仅总结跨页面反复出现的语言质量差异，不猜测来源"
}
```

分项上限：technical_accuracy <= 30，fidelity <= 20，chinese_naturalness <= 20，teaching_expression <= 15，terminology_consistency <= 10，readability <= 5；total = 六项之和。

# zh-CN Glossary

以下为本实验必须遵守的正式 zh-CN glossary；评审时须同时检查 mandatory、preferred、forbidden 和 keep。

````yaml
locale: zh-CN
mandatory:
  A Tour of Go: Go 语言之旅
  previous: 上一页
  next: 下一页
  Run: 运行
  Format: 格式化
  slide: 页面
  slides: 页面
  Go Playground: Go 语言演练场
  constraint: 约束
  type switch: 类型选择
  type switches: 类型选择
  type assertion: 类型断言
  type assertions: 类型断言
  interface value: 接口值
  interface type: 接口类型
  concrete type: 具体类型
  type parameter: 类型参数
  type parameters: 类型参数
preferred:
  Go programming language: Go 编程语言
  built-in: 内置
  channel: 通道
  tour: 教程
  the tour: 本教程
  sandbox: 沙箱
  deterministic output: 确定性输出
  package: 包
  import path: 导入路径
  package name: 包名
  import statement: 导入语句
  exported name: 导出名
  unexported name: 未导出的名称
  standard library: 标准库
  iteration: 迭代
  loop condition: 循环条件
  type inference: 类型推断
  untyped constant: 无类型常量
  untyped numeric constant: 无类型数值常量
  module: 模块
  exercise: 练习
  "syntax \u0068ighlighting": 语法高亮
  map: 映射
  maps: 映射
forbidden:
  - 幻灯片
  - 本之旅
  - 欢迎使用 Go 编程语言之旅
keep:
  - Go
  - gofmt
  - PageUp
  - PageDown
  - Shift
  - Enter
  - Ctrl
  - goroutine
  - goroutines
  - Goroutines
````

# methods/24

## English Source

````article
* Images

[[/pkg/image/#Image][Package image]] defines the `Image` interface:

	package image

	type Image interface {
		ColorModel() color.Model
		Bounds() Rectangle
		At(x, y int) color.Color
	}

*Note*: the `Rectangle` return value of the `Bounds` method is actually an
[[/pkg/image/#Rectangle][`image.Rectangle`]], as the
declaration is inside package `image`.

(See [[/pkg/image/#Image][the documentation]] for all the details.)

The `color.Color` and `color.Model` types are also interfaces, but we'll ignore that by using the predefined implementations `color.RGBA` and `color.RGBAModel`. These interfaces and types are specified by the [[/pkg/image/color/][image/color package]].

.play methods/images.go

````

## Candidate A

````article
* 图像

[[/pkg/image/#Image][image 包]] 定义了 `Image` 接口：

	package image

	type Image interface {
		ColorModel() color.Model
		Bounds() Rectangle
		At(x, y int) color.Color
	}

*注意*： `Rectangle`（ `Bounds` 方法的返回值）实际上是一个
[[/pkg/image/#Rectangle][`image.Rectangle`]]，因为该声明位于 `image` 包中。

（有关全部细节，请参阅 [[/pkg/image/#Image][文档]]。）

`color.Color` 和 `color.Model` 类型也是接口，不过我们将使用预定义的实现 `color.RGBA` 和 `color.RGBAModel`，暂且不考虑这一点。这些接口和类型由 [[/pkg/image/color/][image/color 包]] 定义。

.play methods/images.go
````

## Candidate B

````article
* 图像

[[/pkg/image/#Image][image 包]]定义了 `Image` 接口：

	package image

	type Image interface {
		ColorModel() color.Model
		Bounds() Rectangle
		At(x, y int) color.Color
	}

*注意*： `Rectangle` 类型的 `Bounds` 方法返回值实际上是一个
[[/pkg/image/#Rectangle][`image.Rectangle`]]，因为该声明位于 `image` 包中。

（全部细节请参阅 [[/pkg/image/#Image][文档]]。）

`color.Color` 和 `color.Model` 类型也都是接口，不过这里我们使用预定义的实现 `color.RGBA` 和 `color.RGBAModel`，暂且不考虑这一点。这些接口和类型由 [[/pkg/image/color/][image/color 包]]定义。

.play methods/images.go
````

## Candidate C

````article
* 图像

[[/pkg/image/#Image][图像包]] 定义了 `Image` 接口：

	package image

	type Image interface {
		ColorModel() color.Model
		Bounds() Rectangle
		At(x, y int) color.Color
	}

*注意*： `Rectangle` 是 `Bounds` 方法的返回值，它实际上是
[[/pkg/image/#Rectangle][`image.Rectangle`]]，因为该声明位于 `image` 包中。

（有关全部细节，请参阅 [[/pkg/image/#Image][文档]]。）

`color.Color` 和 `color.Model` 类型也是接口，不过我们将使用预定义的实现 `color.RGBA` 和 `color.RGBAModel`，因而无需考虑这一点。这些接口和类型由 [[/pkg/image/color/][image/color 包]] 定义。

.play methods/images.go
````

# concurrency/7

## English Source

````article
* Exercise: Equivalent Binary Trees

There can be many different binary trees with the same sequence of values stored in it. For example, here are two binary trees storing the sequence 1, 1, 2, 3, 5, 8, 13.

.image /tour/static/img/tree.png

A function to check whether two binary trees store the same sequence is quite complex in most languages. We'll use Go's concurrency and channels to write a simple solution.

This example uses the `tree` package, which defines the type:

	type Tree struct {
		Left  *Tree
		Value int
		Right *Tree
	}


Continue description on [[javascript:click('.next-page')][next page]].

````

## Candidate A

````article
* 练习：等价二叉树

许多不同的二叉树中可以存储相同的值序列。例如，下面两棵二叉树存储的序列都是 1、1、2、3、5、8、13。

.image /tour/static/img/tree.png

在大多数语言中，编写一个函数来检查两棵二叉树是否存储了相同的序列相当复杂。我们将利用 Go 的并发机制和通道来编写一个简单的解决方案。

此示例使用 `tree` 包，该包定义了以下类型：

	type Tree struct {
		Left  *Tree
		Value int
		Right *Tree
	}


请在 [[javascript:click('.next-page')][下一页]] 继续阅读题目说明。
````

## Candidate B

````article
* 练习：等价二叉树

可以有许多不同的二叉树存储着相同的值序列。例如，下面两棵二叉树都存储着序列 1、1、2、3、5、8、13。

.image /tour/static/img/tree.png

在大多数语言中，编写一个检查两棵二叉树是否存储相同序列的函数都相当复杂。我们将利用 Go 的并发机制和通道来编写一个简单的解决方案。

此示例使用 `tree` 包，该包定义了以下类型：

	type Tree struct {
		Left  *Tree
		Value int
		Right *Tree
	}


请在 [[javascript:click('.next-page')][下一页]]继续阅读说明。
````

## Candidate C

````article
* 练习：等价二叉树

许多不同的二叉树都可能存储同一个数值序列。例如，下面两棵二叉树存储的都是序列 1, 1, 2, 3, 5, 8, 13。

.image /tour/static/img/tree.png

在大多数语言中，编写一个函数来检查两棵二叉树是否存储了相同的序列相当复杂。我们将利用 Go 的并发机制和通道编写一个简单的解决方案。

本例使用 `tree` 包，该包定义了如下类型：

	type Tree struct {
		Left  *Tree
		Value int
		Right *Tree
	}


请在 [[javascript:click('.next-page')][下一页]] 继续阅读说明。
````

# concurrency/11

## English Source

````article
* Where to Go from here...


The
[[/doc/][Go Documentation]] is a great place to
start.
It contains references, tutorials, videos, and more.

To learn how to organize and work with Go code, read [[/doc/code][How to Write Go Code]].

If you need help with the standard library, see the [[/pkg/][package reference]]. For help with the language itself, you might be surprised to find the [[/ref/spec][Language Spec]] is quite readable.

To further explore Go's concurrency model, watch
[[https://www.youtube.com/watch?v=f6kdp27TYZs][Go Concurrency Patterns]]
([[/talks/2012/concurrency.slide][slides]])
and
[[https://www.youtube.com/watch?v=QDDwwePbDtw][Advanced Go Concurrency Patterns]]
([[/talks/2013/advconc.slide][slides]])
and read the
[[/doc/codewalk/sharemem/][Share Memory by Communicating]]
codewalk.

To get started writing web applications, watch
[[https://vimeo.com/53221558][A simple programming environment]]
([[/talks/2012/simple.slide][slides]])
and read the
[[/doc/articles/wiki/][Writing Web Applications]] tutorial.

The [[/doc/codewalk/functions/][First Class Functions in Go]] codewalk gives an interesting perspective on Go's function types.

The [[/blog/][Go Blog]] has a large archive of informative Go articles.

Visit [[/][the Go home page]] for more.

````

## Candidate A

````article
* 接下来……

[[/doc/][Go 文档]]是一个很好的起点。
其中包含参考资料、教程、视频等内容。

要了解如何组织和使用 Go 代码，请阅读 [[/doc/code][如何编写 Go 代码]]。

如果你需要标准库方面的帮助，请参阅 [[/pkg/][包参考]]。如果需要了解语言本身，你可能会惊讶地发现 [[/ref/spec][语言规范]]其实相当易读。

要进一步探索 Go 的并发模型，请观看
[[https://www.youtube.com/watch?v=f6kdp27TYZs][Go 并发模式]]
（[[/talks/2012/concurrency.slide][页面]]）
以及
[[https://www.youtube.com/watch?v=QDDwwePbDtw][高级 Go 并发模式]]
（[[/talks/2013/advconc.slide][页面]]），并阅读
[[/doc/codewalk/sharemem/][通过通信共享内存]]
代码导览。

要开始编写 Web 应用，请观看
[[https://vimeo.com/53221558][一个简单的编程环境]]
（[[/talks/2012/simple.slide][页面]]）
并阅读
[[/doc/articles/wiki/][编写 Web 应用]]教程。

[[/doc/codewalk/functions/][Go 中的一等函数]]
代码导览从一个有趣的角度介绍了 Go 的函数类型。

[[/blog/][Go 博客]]收录了大量内容丰富的 Go 文章。

访问 [[/][Go 首页]]了解更多内容。
````

## Candidate B

````article
* 接下来去哪里……


[[/doc/][Go 文档]] 是一个很好的起点。
其中包含参考资料、教程、视频等内容。

要了解如何组织和使用 Go 代码，请阅读 [[/doc/code][如何编写 Go 代码]]。

如果需要标准库方面的帮助，请查阅 [[/pkg/][包参考文档]]。如果需要语言本身的帮助，你可能会惊讶地发现 [[/ref/spec][语言规范]] 相当易读。

要进一步探索 Go 的并发模型，请观看
[[https://www.youtube.com/watch?v=f6kdp27TYZs][Go 并发模式]]
（[[/talks/2012/concurrency.slide][页面]]）
和
[[https://www.youtube.com/watch?v=QDDwwePbDtw][高级 Go 并发模式]]
（[[/talks/2013/advconc.slide][页面]]），
并阅读
[[/doc/codewalk/sharemem/][通过通信共享内存]]
代码导览。

要开始编写 Web 应用，请观看
[[https://vimeo.com/53221558][一个简单的编程环境]]
（[[/talks/2012/simple.slide][页面]]），
并阅读
[[/doc/articles/wiki/][编写 Web 应用]] 教程。

[[/doc/codewalk/functions/][Go 中的一等函数]] 代码导览从一个有趣的视角介绍了 Go 的函数类型。

[[/blog/][Go 博客]] 收录了大量内容丰富的 Go 文章。

请访问 [[/][Go 主页]] 了解更多内容。
````

## Candidate C

````article
* 接下来去哪儿……


[[/doc/][Go 文档]] 是一个很好的起点，其中包含参考资料、教程、视频等内容。

要了解如何组织和使用 Go 代码，请阅读 [[/doc/code][如何编写 Go 代码]]。

如果需要标准库方面的帮助，请参阅 [[/pkg/][包参考]]。如果需要语言本身方面的帮助，你可能会惊讶地发现 [[/ref/spec][语言规范]] 相当易读。

要进一步探索 Go 的并发模型，请观看
[[https://www.youtube.com/watch?v=f6kdp27TYZs][Go 并发模式]]
（[[/talks/2012/concurrency.slide][页面]]）
和
[[https://www.youtube.com/watch?v=QDDwwePbDtw][高级 Go 并发模式]]
（[[/talks/2013/advconc.slide][页面]]），
并阅读
[[/doc/codewalk/sharemem/][通过通信共享内存]]
代码导览。

要开始编写 Web 应用，请观看
[[https://vimeo.com/53221558][一个简单的编程环境]]
（[[/talks/2012/simple.slide][页面]]），
并阅读
[[/doc/articles/wiki/][编写 Web 应用]] 教程。

[[/doc/codewalk/functions/][Go 中的一等函数]] 代码导览从一个有趣的角度介绍了 Go 的函数类型。

[[/blog/][Go 博客]] 收录了大量内容丰富的 Go 文章。

访问 [[/][Go 主页]]，了解更多内容。
````

# generics/1

## English Source

````article
* Type parameters

Go functions can be written to work on multiple types using type parameters. The
type parameters of a function appear between brackets, before the function's
arguments.

  func Index[T comparable](s []T, x T) int

This declaration means that `s` is a slice of any type `T` that fulfills the
built-in constraint `comparable`. `x` is also a value of the same type.

`comparable` is a useful constraint that makes it possible to use the `==` and
`!=` operators on values of the type. In this example, we use it to compare a
value to all slice elements until a match is found. This `Index` function works
for any type that supports comparison.

.play generics/index.go

````

## Candidate A

````article
* 类型参数

可以使用类型参数编写适用于多种类型的 Go 函数。函数的类型参数出现在函数参数之前的方括号中。

  func Index[T comparable](s []T, x T) int

此声明表示， `s` 是任意类型 `T` 的切片，且该类型满足内置约束 `comparable`。 `x` 也是同一类型的值。

`comparable` 是一个实用的约束，允许对该类型的值使用 `==` 和
`!=` 运算符。在此示例中，我们用它将一个值依次与切片中的每个元素进行比较，直到找到匹配项。此 `Index` 函数适用于任何支持比较的类型。

.play generics/index.go
````

## Candidate B

````article
* 类型参数

Go 函数可以通过类型参数编写成适用于多种类型的形式。函数的类型参数位于参数列表之前，并写在方括号中。

  func Index[T comparable](s []T, x T) int

这个声明表示， `s` 是由任意类型 `T` 组成的切片，而该类型必须满足内置约束 `comparable`。 `x` 也是同一类型的值。

`comparable` 是一个很有用的约束，它使我们可以对该类型的值使用 `==` 和
`!=` 运算符。在这个示例中，我们用它将一个值依次与切片中的所有元素比较，直到找到匹配项。这个 `Index` 函数适用于任何支持比较的类型。

.play generics/index.go
````

## Candidate C

````article
* 类型参数

通过使用类型参数，可以编写能够处理多种类型的 Go 函数。函数的
类型参数位于参数之前的方括号内。

  func Index[T comparable](s []T, x T) int

此声明表示， `s` 是任意类型 `T` 的切片，该类型满足
内置约束 `comparable`。 `x` 也是同一类型的值。

`comparable` 是一个实用的约束，它使 `==` 和
`!=` 运算符可用于该类型的值。在本例中，我们用它将一个
值依次与切片中的所有元素比较，直到找到匹配项。 `Index` 函数
适用于任何支持比较的类型。

.play generics/index.go
````

# flowcontrol/8

## English Source

````article
* Exercise: Loops and Functions

As a way to play with functions and loops, let's implement a square root function: given a number x, we want to find the number z for which z² is most nearly x.

Computers typically compute the square root of x using a loop.
Starting with some guess z, we can adjust z based on how close z² is to x,
producing a better guess:

	z -= (z*z - x) / (2*z)

Repeating this adjustment makes the guess better and better
until we reach an answer that is as close to the actual square root as can be.

Implement this in the `func`Sqrt` provided.
A decent starting guess for z is 1, no matter what the input.
To begin with, repeat the calculation 10 times and print each z along the way.
See how close you get to the answer for various values of x (1, 2, 3, ...)
and how quickly the guess improves.

Hint: To declare and initialize a floating point value,
give it floating point syntax or use a conversion:

	z := 1.0
	z := float64(1)

Next, change the loop condition to stop once the value has stopped
changing (or only changes by a very small amount).
See if that's more or fewer than 10 iterations.
Try other initial guesses for z, like x, or x/2.
How close are your function's results to the [[/pkg/math/#Sqrt][math.Sqrt]] in the standard library?

(*Note:* If you are interested in the details of the algorithm, the z² − x above
is how far away z² is from where it needs to be (x), and the division by 2z is the derivative
of z², to scale how much we adjust z by how quickly z² is changing.
This general approach is called [[https://en.wikipedia.org/wiki/Newton%27s_method][Newton's method]].
It works well for many functions but especially well for square root.)

.play flowcontrol/exercise-loops-and-functions.go

````

## Candidate A

````article
* 练习：循环与函数

为了练习函数和循环，我们来实现一个平方根函数：给定一个数 x，找出一个数 z，使 z² 最接近 x。

计算机通常使用循环来计算 x 的平方根。先从某个猜测值 z 开始，再根据 z² 与 x 的接近程度来调整 z，从而得到一个更好的猜测值：

	z -= (z*z - x) / (2*z)

不断重复这一调整，猜测值就会越来越准确，直到得到一个尽可能接近实际平方根的答案。

请在给定的 `func`Sqrt` 中实现此算法。无论输入是什么，z 都可以从一个不错的猜测值 1 开始。首先，重复计算 10 次，并在每次计算时打印 z。观察对于不同的 x 值（1、2、3……），结果能多么接近正确答案，以及猜测值改善得有多快。

提示：要声明并初始化一个浮点值，请使用浮点数语法，或进行类型转换：

	z := 1.0
	z := float64(1)

接下来，更改循环条件，使其在值不再变化（或变化量非常小）时停止。看看所需的迭代次数是多于还是少于 10 次。尝试将 x 或 x/2 等其他值作为 z 的初始猜测值。你的函数结果与标准库中的 [[/pkg/math/#Sqrt][math.Sqrt]] 有多接近？

（*注意：* 如果你对该算法的细节感兴趣，上面的 z² − x 表示 z² 与其目标值（x）相差多少，而除以 2z 是利用 z² 的导数，根据 z² 的变化速度来调整 z 的变化幅度。这种通用方法称为 [[https://en.wikipedia.org/wiki/Newton%27s_method][牛顿法]]。它适用于许多函数，尤其适合计算平方根。）

.play flowcontrol/exercise-loops-and-functions.go
````

## Candidate B

````article
* 练习：循环与函数

为了练习函数和循环，我们来实现一个平方根函数：给定一个数 x，我们希望找到一个数 z，使 z² 尽可能接近 x。

计算机通常使用循环来计算 x 的平方根。
从某个猜测值 z 开始，我们可以根据 z² 与 x 的接近程度调整 z，
从而得到一个更好的猜测值：

	z -= (z*z - x) / (2*z)

不断重复这样的调整，猜测值会越来越准确，
直到得到一个尽可能接近实际平方根的答案。

请在提供的 `func`Sqrt` 中实现这一算法。
无论输入是什么，z 都可以从 1 这个不错的初始猜测值开始。
首先，将计算重复 10 次，并在每次计算过程中打印 z。
观察对于不同的 x 值（1、2、3……），结果能有多接近正确答案，
以及猜测值改善得有多快。

提示：要声明并初始化一个浮点数值，
可以使用浮点数语法，或者进行类型转换：

	z := 1.0
	z := float64(1)

接着，修改循环条件，让它在数值不再发生变化
（或者变化量非常小）时停止。
看看这样需要的迭代次数是多于还是少于 10 次。
还可以尝试其他 z 的初始猜测值，例如 x 或 x/2。
你的函数结果与标准库中的 [[/pkg/math/#Sqrt][math.Sqrt]] 有多接近？

（*注意：* 如果你对这个算法的细节感兴趣，上面的 z² − x
表示 z² 与它应达到的位置（x）相差多少，而除以 2z 是利用
z² 的导数，根据 z² 变化的快慢来调整我们修正 z 的幅度。
这种通用方法称为 [[https://en.wikipedia.org/wiki/Newton%27s_method][牛顿法]]。
它适用于许多函数，而用于计算平方根时尤其有效。）

.play flowcontrol/exercise-loops-and-functions.go
````

## Candidate C

````article
* 练习：循环与函数

为了练习使用函数和循环，让我们实现一个平方根函数：给定数字 x，我们要找出一个数 z，使 z² 尽可能接近 x。

计算机通常使用循环来计算 x 的平方根。
从某个猜测值 z 开始，我们可以根据 z² 与 x 的接近程度调整 z，
从而得到一个更好的猜测值：

	z -= (z*z - x) / (2*z)

不断重复这一调整，猜测值就会越来越准确，
直到得到一个尽可能接近实际平方根的答案。

请在已提供的 `func`Sqrt` 中实现这一算法。
无论输入是什么，z 的一个合理初始猜测值都是 1。
首先，重复计算 10 次，并在过程中打印每次得到的 z。
看看对于 x 的不同取值（1、2、3……），结果与答案有多接近，
以及猜测值改善得有多快。

提示：要声明并初始化一个浮点数值，
请使用浮点数语法，或进行类型转换：

	z := 1.0
	z := float64(1)

接下来，修改循环条件，使其在值不再变化
（或变化量非常小）时停止。
看看这需要多于还是少于 10 次迭代。
尝试使用其他 z 的初始猜测值，例如 x 或 x/2。
你的函数结果与标准库中的 [[/pkg/math/#Sqrt][math.Sqrt]] 有多接近？

（*注意：* 如果你对该算法的细节感兴趣，上面的 z² − x
表示 z² 与其目标值（x）的差距，而除以 2z 是利用 z² 的导数，
根据 z² 的变化速度来调整 z 的改变量。
这种通用方法称为 [[https://en.wikipedia.org/wiki/Newton%27s_method][牛顿法]]。
它适用于许多函数，但尤其适合计算平方根。）

.play flowcontrol/exercise-loops-and-functions.go
````
