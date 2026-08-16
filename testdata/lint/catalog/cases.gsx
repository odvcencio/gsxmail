package lintcatalog

func CaseEM001() Node {
    return <script>alert(1)</script>
}

func CaseEM002() Node {
    return <form></form>
}

func CaseEM003() Node {
    return <blink>hi</blink>
}

func CaseEM004() Node {
    return <div onclick="alert(1)">hi</div>
}

//gosx:island
func CaseEM005() Node {
    return <div>hi</div>
}

func CaseEM006() Node {
    return <link rel="stylesheet" href="x.css" />
}

func CaseEM010(props Props) Node {
    return <email.Signal text={props.Name * 2} />
}

func CaseEM011(props Props) Node {
    return <email.Signal text={props.Items.length} />
}

func CaseEM012(props Props) Node {
    return <email.Signal text={props.Bogus} />
}

func CaseEM013(props Props) Node {
    return <email.Signal text={props.Roster} />
}

func CaseEM014(props Props) Node {
    return <email.Signal text={undeclaredHelper(props.Name)} />
}

func CaseEM015(props Props) Node {
    return <email.Signal text={wrongArity(props.Name)} />
}

func CaseEM020() Node {
    return <Something />
}

func CaseEM030() Node {
    return <If><email.Signal text="x" /></If>
}

func CaseEM031() Node {
    return <If cond={true}>bad text</If>
}

func CaseEM032(props Props) Node {
    return <Each of={props.DraftTime} as="row"><email.Signal text="x" /></Each>
}

func CaseEM033(props Props) Node {
    return <Each of={props.Items}><email.Signal text="x" /></Each>
}

func CaseEM101() Node {
    return <div style="border-radius: 4px;"></div>
}

func CaseEM102() Node {
    return <div style="margin-top: 4px;"></div>
}

func CaseEM103() Node {
    return <div style="not a declaration"></div>
}

func CaseEM104() Node {
    return <div class="foo"></div>
}

func CaseEM110() Node {
    return <a href="javascript:alert(1)">click</a>
}

func CaseEM111() Node {
    return <img src="/relative.png" alt="pic" />
}

func CaseEM112() Node {
    return <img src="https://example.com/pic.png" alt="" />
}
