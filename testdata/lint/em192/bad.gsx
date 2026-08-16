package em192

func BadEmail(props BadProps) Node {
    return <email.Shell wordmark={props.Name} shortCode="X" tagline="T" title="T" lang="en" preheader="p">
        <email.Note text={props.Name} />
    </email.Shell>
}
