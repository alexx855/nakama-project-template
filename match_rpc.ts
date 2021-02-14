// Copyright 2020 The Nakama Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

let rpcFindMatch: nkruntime.RpcFunction = function (ctx: nkruntime.Context, logger: nkruntime.Logger, nk: nkruntime.Nakama, payload: string): string {
    if (!ctx.userId) {
        throw Error('No user ID in context');
    }

    if (!payload) {
        throw Error('Expects payload.');
    }

    let request = {} as RpcFindMatchRequest;
    try {
        request = JSON.parse(payload);
    } catch (error) {
        logger.error('Error parsing json message: %q', error);
        throw error;
    }

    let matches: nkruntime.Match[];
    try {
        const query = `+label.open:1 +label.fast:${request.fast ? 1 : 0}`;
        matches = nk.matchList(10, true, null, null, 1, query);
    } catch (error) {
        logger.error('Error listing matches: %v', error);
        throw error;
    }

    let match_ids: string[] = [];
    if (matches.length > 0) {
        // There are one or more ongoing matches the user could join.
        match_ids = matches.map(m => m.matchId);
    } else {
        // No available matches found, create a new one.
        try {
            match_ids.push(nk.matchCreate(moduleName, { fast: request.fast }));
        } catch (error) {
            logger.error('Error creating match: %v', error);
            throw error;
        }
    }

    // try {
    //     const user =  nk.usersGetId([ctx.userId])[0];
    // } catch (error) {
    //     logger.error('Error listing matches: %v', error);
    //     throw error;
    // }

    let res: RpcFindMatchResponse = { match_ids };
    return JSON.stringify(res);
}

let rpcListMatches: nkruntime.RpcFunction = function (ctx: nkruntime.Context, logger: nkruntime.Logger, nk: nkruntime.Nakama, payload: string): string {
    if (!ctx.userId) {
        throw Error('No user ID in context');
    }

    logger.info('ctx.userId', ctx.userId);

    // if (!payload) {
    //     throw Error('Expects payload.');
    // }

    let matches: nkruntime.Match[];
    try {
        const query = "+label.open:1";
        matches = nk.matchList(10, true, null, null, 1, query);
        logger.info('Matches', matches);
    } catch (error) {
        logger.error('Error listing matches: %v', error);
        throw error;
    }

    let match_ids: string[] = [];
    if (matches.length > 0) {
        // There are one or more ongoing matches the user could join.
        match_ids = matches.map(m => m.matchId);
    }

    let res: RpcFindMatchResponse = { match_ids };
    return JSON.stringify(res);
}

let rpcGeMatch: nkruntime.RpcFunction = function (ctx: nkruntime.Context, logger: nkruntime.Logger, nk: nkruntime.Nakama, payload: string): string {
    if (!ctx.userId) {
        throw Error('No user ID in context');
    }

    logger.info('rpcGeMatch() ctx.userId', ctx.userId);

    if (!payload) {
        throw Error('Expects payload.');
    }

    let state: State | null = null
    let match: Match | null = null;

    try {
        // logger.info('payload', payload);
        const data = JSON.parse(payload);
        match = nk.matchGet(data.matchId);
        state = {
            label: JSON.parse(match.label ? match.label : ''),
            emptyTicks: 0,
            presences: {},
            joinsInProgress: 0,
            playing: false,
            board: [],
            marks: {},
            mark: Mark.UNDEFINED,
            deadlineRemainingTicks: 0,
            winner: null,
            winnerPositions: null,
            nextGameRemainingTicks: 0,
        }

        // &{random:0xc001f9fef0 label:0xc000123b60 emptyTicks:0 presences:map[5bae80d3-6f42-44fb-9543-f3e1dc15b32d:0xc000c0e000] joinsInProgress:0 playing:false board:[] marks:map[] mark:0 deadlineRemainingTicks:0 winner:0 nextGameRemainingTicks:0}
        
    } catch (error) {
        logger.error('Error listing matches: %v', error);
        throw error;
    }

    let res: RpcGetMatchResponse = { match, state };
    return JSON.stringify(res);
}
