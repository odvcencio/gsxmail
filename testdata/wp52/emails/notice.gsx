// notice.gsx is the WP5.2 golden fixture (design spec section 15, WP5.2):
// a small Shell/Headline/CTA/Footer template that renders three shapes —
// preheader-on, hardened+dark (locked), hardened+dark (adaptive) — from
// one source tree, so each new emission's byte diff stays reviewable.
package emails

func Notice(props NoticeProps) Node {
    return <email.Shell
        wordmark={props.Wordmark}
        shortCode={props.ShortCode}
        tagline={props.Tagline}
        title={props.Title}
        lang="en"
        preheader={props.Preheader}>
        <email.Headline title={props.HeadTitle} lede={props.HeadLede} />
        <email.CTA label={props.CTALabel} href={props.CTAHref} />
        <email.Footer signoff={props.FooterSign} note={props.FooterNote} />
    </email.Shell>
}

// NoticeOutlookOff is byte-identical to Notice except its Shell attribute
// outlook="off" overrides the Set's own Options.Outlook at the template
// level (design spec section 15, WP5.2's per-Shell surface). It proves the
// override wins even when the Set default is the hardened "" contract.
func NoticeOutlookOff(props NoticeProps) Node {
    return <email.Shell
        wordmark={props.Wordmark}
        shortCode={props.ShortCode}
        tagline={props.Tagline}
        title={props.Title}
        lang="en"
        outlook="off">
        <email.Headline title={props.HeadTitle} lede={props.HeadLede} />
        <email.CTA label={props.CTALabel} href={props.CTAHref} />
        <email.Footer signoff={props.FooterSign} note={props.FooterNote} />
    </email.Shell>
}

// NoticeOutlookGhostTables is Notice's mirror image: its Shell attribute
// outlook="ghost-tables" overrides a Set default of "off", proving the
// per-Shell surface wins in both directions.
func NoticeOutlookGhostTables(props NoticeProps) Node {
    return <email.Shell
        wordmark={props.Wordmark}
        shortCode={props.ShortCode}
        tagline={props.Tagline}
        title={props.Title}
        lang="en"
        outlook="ghost-tables">
        <email.Headline title={props.HeadTitle} lede={props.HeadLede} />
        <email.CTA label={props.CTALabel} href={props.CTAHref} />
        <email.Footer signoff={props.FooterSign} note={props.FooterNote} />
    </email.Shell>
}
