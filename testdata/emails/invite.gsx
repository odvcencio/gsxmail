package emails

func InviteEmail(props InviteProps) Node {
    return <email.Shell
        wordmark="GRIDIRON 2000"
        shortCode="G2K"
        tagline="DYNASTY FANTASY LEAGUE"
        title="GRIDIRON 2000"
        lang="en">
        <email.Signal text={"DRAFT EVENT // " + props.ShortDate} />
        <email.Headline
            title="YOU'RE IN."
            lede="A seat is holding for you in an eight-manager dynasty league. Aqua vs Orange. Rosters carry over. Receipts are forever." />
        <email.Panel>
            <email.PanelRow label="DRAFT" value={props.LongDate + " · " + props.DraftTime} />
            <email.PanelRow label="VENUE" value="During the Dolphins preseason game — bring both screens." />
            <email.PanelRow label="YOUR KEY" value={"Sign in with Google as " + props.Email} />
        </email.Panel>
        <email.CTA label="CLAIM YOUR SEAT →" href={props.LeagueURL} />
        <email.PickList title="NEXT STEPS //">
            <email.Item>Claim your seat</email.Item>
            <email.Item>Rename your team</email.Item>
            <email.Item>Build your Big Board</email.Item>
            <email.Item>Read the Rules page</email.Item>
        </email.PickList>
        <email.Footer
            signoff="— The Commissioner"
            note="GRIDIRON 2000 · Eight seats. One trophy. Permanent group-chat evidence." />
    </email.Shell>
}
