import { Button } from "antd";
import { useEffect, useRef, useState } from "react";
import styles from "./MenuButton.module.css";

interface MenuButtonProps {
	items: Array<{ label: string; icon?: string; onClick: () => void }>;
	label?: string;
	ariaLabel?: string;
}

export default function MenuButton({
	items,
	label = "⋯⋯",
	ariaLabel = "菜单",
}: MenuButtonProps) {
	const [isOpen, setIsOpen] = useState(false);
	const menuRef = useRef<HTMLDivElement>(null);
	const buttonRef = useRef<HTMLButtonElement>(null);

	useEffect(() => {
		const handleEscape = (e: KeyboardEvent) => {
			if (e.key === "Escape" && isOpen) {
				setIsOpen(false);
				buttonRef.current?.focus();
			}
		};
		const handleClickOutside = (e: MouseEvent) => {
			const target = e.target as Node | null;
			if (
				isOpen &&
				target &&
				!menuRef.current?.contains(target) &&
				!buttonRef.current?.contains(target)
			) {
				setIsOpen(false);
			}
		};
		document.addEventListener("keydown", handleEscape);
		document.addEventListener("mousedown", handleClickOutside);
		return () => {
			document.removeEventListener("keydown", handleEscape);
			document.removeEventListener("mousedown", handleClickOutside);
		};
	}, [isOpen]);

	useEffect(() => {
		if (!isOpen || !menuRef.current) return;

		const focusableElements = menuRef.current.querySelectorAll("button");
		const firstElement = focusableElements[0];
		const lastElement = focusableElements[focusableElements.length - 1];

		const handleTab = (e: KeyboardEvent) => {
			if (e.key !== "Tab") return;

			if (e.shiftKey) {
				if (document.activeElement === firstElement) {
					e.preventDefault();
					lastElement?.focus();
				}
			} else {
				if (document.activeElement === lastElement) {
					e.preventDefault();
					firstElement?.focus();
				}
			}
		};

		menuRef.current.addEventListener("keydown", handleTab);
		return () => menuRef.current?.removeEventListener("keydown", handleTab);
	}, [isOpen]);

	const handleItemClick = (onClick: () => void) => {
		onClick();
		setIsOpen(false);
	};

	return (
		<div className={styles.menuButtonContainer}>
			<Button
				ref={buttonRef}
				type="text"
				size="small"
				onClick={() => setIsOpen(!isOpen)}
				aria-haspopup="true"
				aria-expanded={isOpen}
				aria-label={ariaLabel}
				className={styles.menuButton}
			>
				{label}
			</Button>

			{isOpen && (
				<div
					ref={menuRef}
					className={styles.dropdown}
					role="menu"
					aria-label={ariaLabel}
				>
					{items.map((item) => (
						<button
							key={item.label}
							type="button"
							className={styles.menuItem}
							role="menuitem"
							onClick={() => handleItemClick(item.onClick)}
						>
							{item.icon} {item.label}
						</button>
					))}
				</div>
			)}
		</div>
	);
}
