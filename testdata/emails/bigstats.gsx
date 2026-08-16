package emails

func BigStats(props BigStatsProps) Node {
    return <email.Shell
        wordmark={props.Wordmark}
        shortCode="BIG"
        tagline="SIZE BUDGET TEST"
        title="BIG"
        lang="en">
        <email.StatTable title="ROWS //" header={props.Header}>
            <Each of={props.Rows} as="row">
                <email.StatRow cells={row.Cells} />
            </Each>
        </email.StatTable>
    </email.Shell>
}
