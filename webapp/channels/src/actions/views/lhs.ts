// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {selectChannel} from 'mattermost-redux/actions/channels';
import {DispatchFunc, GetStateFunc} from 'mattermost-redux/types/actions';
import {getCurrentRelativeTeamUrl} from 'mattermost-redux/selectors/entities/teams';

import {GlobalState} from 'types/store';
import {LhsItemType} from 'types/store/lhs';
import Constants, {ActionTypes} from 'utils/constants';
import {getHistory} from 'utils/browser_history';
import {SidebarSize} from 'components/resizable_sidebar/constants';

export const setLhsSize = (sidebarSize?: SidebarSize) => {
    let newSidebarSize = sidebarSize;
    if (!sidebarSize) {
        const width = window.innerWidth;

        switch (true) {
        case width <= Constants.SMALL_SIDEBAR_BREAKPOINT: {
            newSidebarSize = SidebarSize.SMALL;
            break;
        }
        case width > Constants.SMALL_SIDEBAR_BREAKPOINT && width <= Constants.MEDIUM_SIDEBAR_BREAKPOINT: {
            newSidebarSize = SidebarSize.MEDIUM;
            break;
        }
        case width > Constants.MEDIUM_SIDEBAR_BREAKPOINT && width <= Constants.LARGE_SIDEBAR_BREAKPOINT: {
            newSidebarSize = SidebarSize.LARGE;
            break;
        }
        default: {
            newSidebarSize = SidebarSize.XLARGE;
        }
        }
    }
    return {
        type: ActionTypes.SET_LHS_SIZE,
        size: newSidebarSize,
    };
};

export const toggle = () => ({
    type: ActionTypes.TOGGLE_LHS,
});

export const open = () => ({
    type: ActionTypes.OPEN_LHS,
});

export const close = () => ({
    type: ActionTypes.CLOSE_LHS,
});

export const selectStaticPage = (itemId: string) => ({
    type: ActionTypes.SELECT_STATIC_PAGE,
    data: itemId,
});

export const selectLhsItem = (type: LhsItemType, id?: string) => {
    return (dispatch: DispatchFunc) => {
        switch (type) {
        case LhsItemType.Channel:
            dispatch(selectChannel(id || ''));
            dispatch(selectStaticPage(''));
            break;
        case LhsItemType.Page:
            dispatch(selectChannel(''));
            dispatch(selectStaticPage(id || ''));
            break;
        case LhsItemType.None:
            dispatch(selectChannel(''));
            dispatch(selectStaticPage(''));
            break;
        default:
            throw new Error('Unknown LHS item type: ' + type);
        }
    };
};

export function switchToLhsStaticPage(id: string) {
    return (dispatch: DispatchFunc, getState: GetStateFunc) => {
        const state = getState() as GlobalState;
        const teamUrl = getCurrentRelativeTeamUrl(state);
        getHistory().push(`${teamUrl}/${id}`);

        return {data: true};
    };
}
