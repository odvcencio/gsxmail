package emails

func WelcomeEmail(props WelcomeProps) Node {
    return <email.Shell
        wordmark={props.Product}
        shortCode="OK"
        tagline="ACCOUNT CREATED"
        title={props.Product}
        lang="en"
        preheader={"Your " + props.Product + " account is ready — sign in to finish setup."}>
        <email.Signal text={"WELCOME // " + props.Name} />
        <email.Headline
            title="YOU'RE IN."
            lede="Your account is live. Sign in to finish setting up your workspace." />
        <email.Panel>
            <email.PanelRow label="ACCOUNT" value={props.Name} />
            <email.PanelRow label="PRODUCT" value={props.Product} />
        </email.Panel>
        <email.CTA label="SIGN IN →" href={props.LoginURL} />
        <email.PickList title="NEXT STEPS">
            <email.Item>Confirm your email address</email.Item>
            <email.Item>Invite your team</email.Item>
            <email.Item>Read the getting-started guide</email.Item>
        </email.PickList>
        <email.Footer
            signoff="— The Team"
            note="You are receiving this email because an account was created with this address." />
    </email.Shell>
}
