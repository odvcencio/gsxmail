// receipt.gsx is the gallery's complete worked example (pixel dossier
// section 8.3), reproduced from the dossier's own .gsx source: a PAID
// Badge, a StatTable built from <Each> over line items, a Panel of
// totals, and a Button (variant="primary" — email.CTA's own alias).
package receipt

func ReceiptEmail(props ReceiptProps) Node {
    return <email.Shell
        wordmark={props.Product}
        shortCode="ACM"
        tagline="ORDER RECEIPT"
        title={props.Product + " receipt"}
        lang="en"
        preheader={"Receipt for order " + props.OrderID + " — total " + props.Total}>
        <email.Badge text="PAID" tone="positive" />
        <email.Headline
            title="PAYMENT RECEIVED."
            lede={"Order " + props.OrderID + ", issued " + props.IssuedOn + ". A copy is attached to your account."} />
        <email.StatTable title="ITEMS //" header={props.Header}>
            <Each of={props.Items} as="item">
                <email.StatRow cells={item.Cells} />
            </Each>
        </email.StatTable>
        <email.Panel>
            <email.PanelRow label="SUBTOTAL" value={props.Subtotal} />
            <email.PanelRow label="TAX" value={props.Tax} />
            <email.PanelRow label="TOTAL" value={props.Total} />
            <email.PanelRow label="BILLED TO" value={props.BilledTo} />
        </email.Panel>
        <email.Button variant="primary" label="VIEW RECEIPT →" href={props.ReceiptURL} />
        <email.Footer
            signoff="— ACME Billing"
            note={props.Product + " · This is a receipt, not an invoice. Keep it for your records."} />
    </email.Shell>
}
