import "./Toggle.css"
import { classNames } from "../util/classNames";


interface ToggleProps {
    name: string
    onChange: any
    on?: string
    off?: string
    checked?: boolean
    disabled?: boolean
    className?: string
}

export function Toggle(data: ToggleProps) {


    return (
        <div className={classNames("toggle-switch", data.className)}>
            <input
                type="checkbox"
                name={data.name}
                id={data.name}
                className="toggle-switch-checkbox"
                onChange={data.onChange}
                checked={data.checked}
                disabled={data.disabled}
            />
            <label className="toggle-switch-label" htmlFor={data.name}>
                <span className="toggle-switch-inner" data-yes={data.on || "Yes"} data-no={data.off || "No"}/>
                <span className="toggle-switch-switch" />
            </label>
        </div>
    )
}