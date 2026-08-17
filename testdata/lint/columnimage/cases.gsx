package columnimage

// CaseRelativeSrc is m4's own EM111 fixture: email.Column's imgSrc must
// be an absolute https URL, the same rule raw <img> and email.Hero
// already enforce.
func CaseRelativeSrc() Node {
    return <email.Columns>
        <email.Column imgSrc="/relative.png" imgAlt="pic" imgWidth="100" imgHeight="100" title="a" text="b" />
        <email.Column title="c" text="d" />
    </email.Columns>
}

// CaseEmptyAlt is m4's own EM112 fixture: email.Column's imgAlt must be
// non-empty when imgSrc is set.
func CaseEmptyAlt() Node {
    return <email.Columns>
        <email.Column imgSrc="https://example.com/pic.png" imgAlt="" imgWidth="100" imgHeight="100" title="a" text="b" />
        <email.Column title="c" text="d" />
    </email.Columns>
}

// CaseNoImage is m4's own negative case: a Column with no image at all
// (no imgSrc) must not trip EM111/EM112 — Column's image is optional.
func CaseNoImage() Node {
    return <email.Columns>
        <email.Column title="a" text="b" />
        <email.Column title="c" text="d" />
    </email.Columns>
}
