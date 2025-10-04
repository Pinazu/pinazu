import BellIcon from '../icons/BellIcon.svelte';
import DocumentsIcon2 from '../icons/DocumentsIcon2.svelte';
import LockIcon from '../icons/LockIcon.svelte';
import ManageToolIcon from '../icons/ManageToolIcon.svelte';
import UserIcon from '../icons/UserIcon.svelte';

// Define the type for the navigation items
export interface NavItem {
	label: string;
	icon: any;
}

// Define the navigation items
export const SETTING_NAV: Record<string, NavItem> = {
	"General": {
		label: "General",
		icon: ManageToolIcon
	},
	"Notifications": {
		label: "Notifications",
		icon: BellIcon
	},
	"Security": {
		label: "Security",
		icon: LockIcon
	},
	"Account": {
		label: "Account",
		icon: UserIcon
	},
	"About": {
		label: "About",
		icon: DocumentsIcon2
	}
}

