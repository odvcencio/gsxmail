// magiclink.gsx is the gallery's sign-in-code template (pixel dossier
// section 8.1): Shell, Headline, Panel (mono OTP row), Note (expiry),
// Button. Short-lived-link wording throughout: the code is a fallback
// for a client that clips the button link.
package magiclink

func MagicLinkEmail(props MagicLinkProps) Node {
    return <email.Shell
        wordmark={props.Product}
        shortCode="IN"
        tagline="SIGN-IN CODE"
        title={props.Product + " sign-in code"}
        lang="en"
        preheader={"Your " + props.Product + " sign-in code: " + props.Code}>
        <email.Headline
            title="YOUR SIGN-IN CODE."
            lede={"Use this code to finish signing in to " + props.Product + " as " + props.Email + "."} />
        <email.Panel>
            <email.PanelRow label="CODE" value={props.Code} />
        </email.Panel>
        <email.Note text={props.ExpiryNote} />
        <email.Button variant="primary" label="SIGN IN →" href={props.LoginURL} />
    </email.Shell>
}
