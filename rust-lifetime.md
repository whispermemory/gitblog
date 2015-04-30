Title: Rust Means Never Having to Close a Socket
Date: 2015-04-26
Category: rust
Tag: 翻译


翻译: By rustup(欢迎转载分享,请注明出处)

原文: <http://blog.skylight.io/rust-means-never-having-to-close-a-socket>

Rust 语言最酷炫的功能是它能够在安全(无段错误)和高性能的同时,仍然能够为使用者进行自动资源管理.
作为一种与众不同的语言, 我们在此讨论的问题可能会很晦涩难懂. 主要围绕一下几点:

+  Rust 和使用带垃圾自动回收机制的语言相同的地方: 永远不需要显式的进行内存释放.
+  Rust 不同于垃圾自动回收机制的语言地方: 永远(注1)不需要显式对文件,套接字和锁进行释放. 
+  Rust 这两种特性的实现, 不会引发运行时开销, 也不牺牲安全性.

如果您曾经泄漏了文件或者套接字,或是使用某种抽象机制泄漏资源,你就会意识到这是一个很大的问题.
你期望有某种机制能免除这种"悬垂资源"的内存漏洞,同时也不用担心在显式关闭套接字出现问题. 本文告诉你一个双全法.

如果你有垃圾回收语言使用经历,你应当关注本文的资源管理方面. 如果你使用的是c/c++这类抽象层次较低的语言,你可能对安全方面的内容更感兴趣.
Rust 集大部分语言的优点,并严格保证这些优秀功能的可使用性. 实际上,这样梦幻般的提升了语言的全面性.

## ownership 系统 ##
Rust 的 ownership 系统是 Rust 能够得以安全的进行自动资源管理的根本.每当用户创建一个对象,它将被创建它的上下文(scope)持有.
下面的例子中的函数将输入文件拷贝入临时文件,处理并将结果拷贝入输出文件
```
fn process(from: &Path, to: &Path) -> IoResult<()> {
    // creates a new tempdir with the specified suffix
        let tempdir = try!(TempDir::new("skylight"));

    // open the input file
        let mut from_file = try!(File::open(from));

    // create a temporary file inside the tempdir
        let mut tempfile =
                try!(File::create(&tempdir.path().join("tmp1")));

    // copy the input file into the tempfile
        try!(io::util::copy(&mut from_file, &mut tempfile));

    // use an external program to process the tmpfile in place

    // after processing, copy the tempfile into the output file
        let mut out = try!(File::create(to));

    io::util::copy(&mut tempfile, &mut out)
    }

```
该例子中 process 函数生成 Tempdir 并一直拥有它的 ownership, 当 process 函数调用结束, process 函数生存周期结束,并自动调用函数销毁 Tempdir 对象.
这是一个简单的自动资源管理的例子,Tempdir 不仅仅代表内存,更是一种托管的资源.一旦程序结束使用它,它的删除逻辑就会执行.

Rust 中几乎所有的资源都会这样,内存自动管理机制方便了对内存释放, 同时资源自动管理机制方便我们对资源进行释放.

旁白:这就是传说中的 RAII(Resource Acquisition Is Initialization) 也是 c++ 中一个很有用处但让人头大疼的概念.
对我而言,许多在手动内存管理方面有效果的技巧都为手动内存管理带来了复杂度. 在一些高级语言中, 不需要释放内存,但是却需要经常的对文件,套接字和锁进行释放.

事实上在 gc 语言中忘记释放这些资源是巨常见的事情. 然而在 Rust 中忘掉关掉套接字和忘记释放内存资源都是小事一桩.程序员也不用为使用悬垂内存和内存泄漏担忧了.
听起来很魔幻不是吗,它到底是耍了什么花样了呢?

首先 Rust 整个 ownership 系统建立在一个对象同时只能有一个 owner 基础上. 怎么确认没有在多个地方引用了 Tempdir 呢? Rust 对象在自己的上下文中被创建出来. 这个上下文将对象的 ownership 传递给别的上下文. 或者拥有该对象一直到上下文结束.每当上下文结束时, Rust 会销毁该上下文所拥有的所有对象.

由于对象的在同一时间内只能被一个生命周期占据, 可以明确判断该对象结束运行时会在哪里被销毁.

```
    struct Person {
    first: String,
        last: String
        }

    fn hello() {
    let yehuda = Person {
        first: "Yehuda".to_string(),
        last: "Katz".to_string()
    };

    // `yehuda` is transferred to `name_size`, so it cannot be
    // used anymore in this function, and it will not be destroyed
    // when this function returns. It is up to `name_size`,
    // or possibly a future owner, to destroy it.
    let size = name_size(yehuda);

    let tom = Person {
        first: "Tom".to_string(),
        last: "Dale".to_string()
    };

    // `tom` wasn't transferred, so it will be
    // destroyed when this function returns.
    }

fn name_size(person: Person) -> uint {
    let Person { first, last } = person;
        first.len() + last.len()

    // this function owns Person, so the Person is destroyed when `name_size` returns
    }

```
对比下这两个函数,yehuda 的 ownership 被传递进了 name_size,当 name_size 的上下文结束后, yehuda 会被销毁. tom 对象则在 hello 函数的上下文结束后销毁.

怎么解释创建临时文件的例子呢, 在 process 函数的第三行,调用了 tempdir.path() 函数. 这是不是意味着有两个生命周期同时指向 Tempdir .还是说 Tempdir 的生命周期转移到了 path 函数内部. 是 process 还是 path 函数在返回的时刻销毁 Tempdir 对象呢.显然不是这样的.

## Borrowing 和 Lending ##
到底发生了什么呢? 看看 path 这个函数的声明
```
fn path(&self) -> &Path
```
这个函数的读法是: path 方法借用自己并返回一个被借用对象.
该函数借用对象,但是并不持有该对象的 ownership, 也不会在函数返回的时候销毁这个对象.它仅仅是在函数调用的过程中使用了这个对象.不能这样使用:创建线程并在新的线程里面使用它.简言之,借用对象(borrowed object)在持有它的函数结束之后仍然存活.

这意味着 Rust 编译器能够分析所有的函数调用, 也能够在编译时知道代码是否持有对象的 ownership ,一旦 ownership 转交, 原始的持有者将被禁访问该对象.

```
struct Person {
    first: String,
    last: String,
    age: uint
}

fn hello() {
    let person = Person {
    first: "Yehuda".to_string(),
    last: "Katz".to_string(),
    age: 32
    };
    
    let thirties = is_thirties(person);
    println!("{}, thirties: {}", person, thirties);
}

// This function tries to take ownership of `Person`; it does not
// ask to borrow it by taking &Person
fn is_thirties(person: Person) {
    person.age >= 30 && person.age < 40
}
```


尝试编译改程序, 会得到下面的错误(删节版本)
```
move.rs:16:34: 16:40 error: use of moved value: `person`
move.rs:16     println!("{}, thirties: {}", person, thirties);
                                            ^~~~~~

move.rs:15:32: 15:38 note: `person` moved here
move.rs:15     let thirties = is_thirties(person);
                                          ^~~~~~

```

这意味着 hello 函数是 Person 对象的初始持有者, 当 is_thirties 函数调用的时候, hello 的 ownership 转移进了 is_thirties 函数. is_thirties 作为新的持有者,会在函数返回的时候释放 Person 对象.

更多时候是使用 borrowing 和 lending 来实现该程序

```
fn hello() {
    let person = Person {
        first: "Yehuda".to_string(),
        last: "Katz".to_string(),
        age: 32
    };

    // lend the person -- don't transfer ownership
    let thirties = is_thirties(&person);

    // now this scope still owns the person
    println!("{}, thirties: {}", person, thirties);
}

fn is_thirties(person: &Person) {
    person.age >= 30 && person.age < 40
}
```

通过上面的例子基本上可以认为,  ownership 检查是函数接口的一部分. Rust 使用者会称之为 'borrow checker', 然而这其中的实现是比较复杂的.

实际大多数时间函数都只是 'borrowing' 了对象, 这时候这种机制很适用. 函数使用了一个对象的值,进行操作然后返回. 在函数返回之后持有该对象更长时间,比如使用了线程, 是比较不常见的,也应该慎重的考虑当前的做法是否适当.

所以应当从一开始在考虑要些一个新函数的时候, 就要使用 borrow 参数而不是使用转移 ownership 的方式. 这样久之成为一种思维习惯. 如果编译器报错(这种情况少之又少只要你遵循这个规则,之后编程一种本能), 那就意味着你在做一些嫉妒危险的事情, 也是你应当仔细思考的时候.

## 从 Borrow 对象中返回 Borrow 字段 ##
让我们看看早些时候的那个函数
```
fn path(&self) -> &Path
```

上文提到, 当一个函数借用了一个对象, 它只能在函数执行的时候使用这个对象. 但是这个函数明显违背该原则并返回该对象.
该函数能够正常运行的原因是, path 函数的调用着明显拥有该对象, 实际上正是调用 path 的函数把 Tempfile 对象借用给了 path 函数. 这种情况下, Rust 编译器回保证返回的 Path(Tempdir 对象) 不会比 Tempfile 的生存期更长

这意味着你可以返回从调用函数 borrowed 的内容, Rust 回仔细的追踪检查原始的持有者. 看一个例子：
```
fn hello() -> &str {
    let person = Person {
        first: "Yehuda".to_string(),
        last: "Katz".to_string(),
        age: 32
    };

    first_name(&person)
}

fn first_name(person: &Person) -> &str {
    // as_slice borrows a slice "view" out of a string
    person.first.as_slice()
}
```
        
看到代码你可以迅速分析出一个问题: hello 函数返回了一个 borrowed &str 对象,但是 hello 本身是持有它内容的 Person 对象的创建者. 只要 Person 不再存在, borrow 的内容（&str) 就会只想一个非法的地址.
如果编译下代码, 会报下面的错误
```
move.rs:8:15: 8:19 error: missing lifetime specifier [E0106]
move.rs:8 fn hello() -> &str {
                        ^~~~
```

出错信息有些晦涩,它暗含着我们尝试返回 borrowed 字节, 但是调用 hello 函数的函数并没有借出 Person 对象.  Rust 希望我们能够告诉它 &str 的原始生命周期在哪儿.

Rust 绑定返回值的生存周期到借用的形参.这里并没有使用借用形参,因此 Rust 要求我们显式声明返回值的生命周期.

实际上, 这意味着函数能够很容易就返回 borrow 形参里面包含的内容. 否则你需要检查函数调用着是否有权限访问返回的值.或者返回克隆的值给他的调用者,以便让调用函数拥有返回克隆对象的 ownership.

## 工效 ##
初见之下,所有的地方都要使用到 ownership 机制, 这样会影响使用 Rust 的工作效率.事实上也确实如此.

但是有很多因素, 让这样做带来的好处远大于表面上看来引入的复杂度.

首先,一大堆真实世界代码都回符合这种 borrow/lend 模型. 写了越来越多的 Rust 代码之后, 我突然意识到 Ruby 这和些 Ruby 的代码具有一样的模式: 函数制造对象, 传递给子函数.子函数操作具体的工作, 然后返回.

这样,反反复复, 这种区别在(使用传递进来参数的函数和长时间持有对象的函数) Rust 就会很明显. 区分了函数签名,检查错误, 就可以获得 Rust 提供的安全机制.

和 C++ 相比, C++ 在大部分场合会显式的进行区分, 但是不进行错误检查. GC 类型的语言不会区分 'transfered' 和 'lent' 两种形参机制.

这意味着 Rust 程序员回很快的学将 borrowing 作为默认的函数声明方式. 这样会大幅度减轻系统的运行负载.

第二, 使用 Rust 一段时间后, 大部分人回发现 borrow checker 错误实际上是警告他们一些真正存在的严重的却又很难发现的问题. 之后 borrow checker 自然的推动你的走在更难犯错误的编程模型道路上.

第三, 我个人觉得, 对对象有清晰的 ownership 会提高对程序的逻辑理解. 普遍的降低引入灾难性的难追踪内存泄漏的可能性.

最后, 自动的资源管理机制, 一方面提高了效率, 一方面又能保证不发生资源泄漏(在我懒得调用释放函数的时候), 也不会烧坏水壶(在我不小心多次调用释放函数的时候).


除了 C++ 程序员外之外,很少有程序员会擅长处理自动资源管理机制.而且这种机制经常把我们揍得嚎啕大哭, 以及让我们觉得它并不是特别实用的机制. Rust 改变了自动资源管理传统机制需要折衷的做法, 并极力的移走你脑海里面的那个低低埋怨的声音: 如果我不需要它, 它对我又有什么用呢'.

## 引用计数(和 GC)  ##

你可能会意识到 Rust 有引用计数指针(并且计划在未来引入 GC 特性)

但是引用计数和 GC 怎么才能和自动资源管理相协调呢?

以我的经验看来, 只要你习惯了使用 ownership 范式, 就很少使用会需要使用 Rc 指针. 比如整个 Cargo 核心代码没有引用计数指针, 仅仅只有一个自动引用计数指针(为了并行编译在多线程之间共享锁)

我认为 ownership 和 lending 很好的映射了大部分真是编程模型. 有一些场景无法适用, 但一般稍微的改进代码结构就能够编译. 这样, 在内存和资源上的泄漏很难发生,代码的清晰度也会提高.

如果不是这样的话, 我猜测有经验的 Rust 开发者也会经常的使用 Rc .

总而言之, 还是回有一些场景需要引用计数, 甚至是 gc. Rust 的智能指针系统允许 Rc 指针透明操作一些 ownership 和 borrowing 系统. 当引用计数减为 0 析构器会自动执行.(这样回有明显的自动变量推导和运行时性能消耗)

## 其它语言中的机制 ##

垃圾自动回收语言经常回提供一些措施帮助程序员处理手动资源管理问题. 在许多现代语言中, 并不需要显式的调用 close 函数, 但是需要在语言构造资源的时候绑定到上下文, 并在操作完毕释放.
看一些例子, 然后讨论下这种做法的劣势.

在 Ruby 中, 可以使用块来指定资源使用的域.一旦块返回, 资源就回清理掉.

```
File.open("/etc/passwd") do |file|
# use the file
end

```

Python 中使用 with 关键字创建约定获取资源并在块执行完毕之后进行释放.

```
with open("/etc/passwd") as file:
  # use the file
```

Ruby使用了通用的语言构造,Python 创建了一种约定,抽象出指定资源关闭机制. 用户不用需要知道真正关闭的方式,但是必须使用特定的语法抽象保证该关闭执行

Go 语言使用 defer 关键字提供在创建资源时指定资源清理逻辑.

```
file, error := os.Open("/etc/passwd")
if err != nil {
    return;
}
defer file.Close()

// use the file
```

这相比使用 try/catch/finally 更优越, 因为能够可以在资源创建的代码附近指定清理逻辑,但并没有对关闭逻辑进行封装.

以上这些方法都有许多问题. 因此极力的推荐你放弃那些 '在实际中资源管理问题最终都不是问题' 的误解.

+  几乎不可能对已经存在的结构事后添加添加资源释放逻辑, 因为客户端已经使用普通的的创建资源 API. 这样一来从上层对象中提取资源变得很困难, 因为资源很可能已经在 API 中泄漏了.

+  基于 block 的方法(Ruby 和 Python, 不包含 Go), 带来了右倾(rightward drift)的问题. 每次处理资源的时候, 都必须创建一个新的上下文( scope ).这样的处理在 Ruby(Ruby 有很好的 blocks机制) 和 Python(有语言级别的上下文构造)很烦, 在 JavaScript 中会引入一个严重的问题, 因为引入新的上下文 ( scope ) 回妨碍返回以及循环退出.

+  这些方法(包括 Go defer)需要为资源的处理生成指定的词法上下文. 每当需要传递资源到多个函数的时候, 这样的编程方式就会不便利甚至不可实现. 实际上这只是在没有完美的对象管理机制的时候,强制使用基于上下文的 ownership 系统. 一旦开始调用其它使用了该资源的方法,当最初的函数关闭资源的时候.这些调用它的函数函数就回挂掉在使用资源的地方,很容易就陷入琐碎的的悬垂资源的的漏洞.

Rust 的自动资源管理中几乎解决了所有的这些问题

+  资源管理对象能够定义一个析构函数, 它会涵括释放逻辑. 使用正常的创建对象创建对象, 析构函数就会在正确的时间点调用. 对象能够在创建之后添加析构函数, 不用修改上层代码.注意析构函数跟GC语言中的终止器(finializer)不同. 他们是可预料的,尤其是对象不再使用的时候.也不会引入运行时开销 --除了析构函数的调用开销.

+  自动资源管理和自动内存管理机制一样, 没有缩进要求.少了烦心,保护周围语意不变.

+  在 Rust 中, 可以像任何其他对象一样传递资源.如果一觉了 ownership 到一个新的上下文(scope), 当新的上下文(scope)结束的时候资源就会关闭. 其它情况下, borrowing 系统回保证不会出现悬垂资源和悬垂内存.

简言之,这种使用内存和资源的系统回带来很多真实的好处.

Rust 的 ownership 用起来不会像垃圾回收那样无脑简单. 我只是说 Rust 智能的减少了很多消耗,甚至在一些情况下工作效能上比垃圾回收语言有所提高.
作为学习它的回报, 你得到了一个足够快,能够直接操作内存, 并保证完全安全的语言. 正因为如此, 使用它能够让所有的高级语言用户进行底层代码开发. 这是令我着迷的. 交流使我们从彼此那里学到更多的东西.

(原文广告)你想学习如何提高 Rails app 的性能吗, 注册 30 天免费使用的 [Skylight](https://www.skylight.io "Skylight") 账号, 不需要银行卡.

注: 当我说 '永远' 的时候, 我指的是非常非常少有的情况. 在垃圾回收语言中,有时会需要直接管理内存. 在 Rust, 偶尔会直接处理内存. 这两种情况,语言代为管理资源是主流,也是一种重要的编程方式.

