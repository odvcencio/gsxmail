// welcome.gsx is the gallery's onboarding template (pixel dossier section
// 8.1): Shell, Headline, PickList, Button, Footer.
package welcome

func WelcomeEmail(props WelcomeProps) Node {
    return <email.Shell
        wordmark={props.Product}
        shortCode="OK"
        tagline="ACCOUNT CREATED"
        title={props.Product}
        lang="en"
        preheader={"Welcome to " + props.Product + " — your account is ready"}>
        <email.Headline
            title="YOU'RE IN."
            lede={"Your " + props.Product + " account is live, " + props.Name + ". Sign in to finish setting up your workspace."} />
        <email.PickList title="NEXT STEPS">
            <email.Item>Confirm your email address</email.Item>
            <email.Item>Invite your team</email.Item>
            <email.Item>Read the getting-started guide</email.Item>
        </email.PickList>
        <email.Button variant="primary" label="SIGN IN →" href={props.LoginURL} />
        <email.Footer
            signoff="— The Team"
            note={"You are receiving this email because your " + props.Product + " account was just created."} />
    </email.Shell>
}
