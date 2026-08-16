package newcomer

// WelcomeEmail is the launch-gate B4 finding's own reproduction fixture: a
// newcomer's first template, with the four attribute mistakes the examiner
// found gsxmail's check used to pass through silently — heading= instead
// of title=, a CTA with no href at all, and a Button whose label/href
// attributes are capitalized wrong (Label/HREF instead of label/href).
func WelcomeEmail(props WelcomeProps) Node {
    return <email.Shell
        wordmark={props.Product}
        shortCode="OK"
        tagline="ACCOUNT CREATED"
        title={props.Product}
        preheader="hi"
        lang="en">
        <email.Headline heading="TYPO" lede="body" />
        <email.CTA label="NO HREF HERE" />
        <email.Button variant="secondary" Label="CapitalL" HREF="https://x.example/" />
        <email.Footer signoff="bye" />
    </email.Shell>
}
